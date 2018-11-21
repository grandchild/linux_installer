package linux_installer

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"regexp"
	"sort"
	"text/template"

	"github.com/cloudfoundry/jibber_jabber"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v2"
)

const (
	DefaultLanguage string = "en"
	displayKey             = "_language_display"
)

type (
	StringMap  map[string]string
	Translator struct {
		language    string
		langStrings map[string]StringMap
		variables   StringMap
		display     string
	}
)

func TranslatorNew() Translator {
	return TranslatorVarNew(StringMap{})
}
func TranslatorVarNew(variables StringMap) Translator {
	languageFiles := MustGetResourceFiltered("languages", regexp.MustCompile(`\.ya?ml$`))
	languages := make(map[string]StringMap)
	for filename, content := range languageFiles {
		languageTag := regexp.MustCompile(`.*/([^/]+)\.ya?ml`).ReplaceAllString(filename, "$1")
		langStrings := make(StringMap)
		err := yaml.Unmarshal([]byte(content), langStrings)
		if err != nil {
			log.Printf("Unable to parse language file %s\n", filename)
			continue
		}
		languages[languageTag] = langStrings
	}
	t := Translator{
		langStrings: languages,
		variables:   variables,
	}
	err := t.SetLanguage(t.getLocale())
	if err != nil {
		err = t.SetLanguage(DefaultLanguage)
		if err != nil {
			panic(err)
		}
	}
	return t
}

func (t *Translator) Get(key string) string {
	variables := make(map[string]string)
	for key, value := range t.variables {
		variables[key] = t.expandVariables(value, t.langStrings[t.language])
	}
	str := t.GetRaw(key)
	return t.expandVariables(str, variables)
}

func (t *Translator) GetRaw(key string) string {
	if langStrings, ok := t.langStrings[t.language]; ok {
		if value, ok := langStrings[key]; ok {
			return value
		}
	}
	if langStrings, ok := t.langStrings[DefaultLanguage]; ok {
		if value, ok := langStrings[key]; ok {
			return value
		}
	}
	return ""
}

func (t *Translator) GetLanguage() string { return t.language }
func (t *Translator) GetLanguages() (languages []string) {
	hasDefault := false
	for lang := range t.langStrings {
		if lang != DefaultLanguage {
			languages = append(languages, lang)
		} else {
			hasDefault = true
		}
	}
	sort.Strings(languages)
	if hasDefault {
		// make sure the default language is first
		languages = append([]string{DefaultLanguage}, languages...) // a.k.a. prepend
	}
	return languages
}

func (t *Translator) GetAllStrings() StringMap { return t.langStrings[t.language] }
func (t *Translator) GetAllVersionsList(key string) (versions []string) {
	languages := t.GetLanguages()
	for _, lang := range languages {
		if value, ok := t.langStrings[lang][key]; ok {
			versions = append(versions, value)
		}
	}
	return versions
}
func (t *Translator) GetAllVersions(key string) StringMap {
	versions := make(StringMap)
	for _, lang := range t.GetLanguages() {
		if value, ok := t.langStrings[lang][key]; ok {
			versions[lang] = value
		} else {
			versions[lang] = ""
		}
	}
	return versions
}

func (t *Translator) GetLanguageOptions() (displayStrings []string) {
	for lang := range t.langStrings {
		displayStrings = append(displayStrings, t.langStrings[lang][displayKey])
	}
	return displayStrings
}
func (t *Translator) GetLanguageOptionKeys() (languageKeys []string) {
	for lang := range t.langStrings {
		languageKeys = append(languageKeys, lang)
	}
	return languageKeys
}

func (t *Translator) SetLanguage(language string) error {
	if langStrings, ok := t.langStrings[language]; ok {
		t.language = language
		t.display = langStrings[displayKey]
		return nil
	}
	return errors.New(fmt.Sprintf("No language '%s'.", language))
}

func (t *Translator) getLocale() string {
	languageTags := []language.Tag{language.Raw.Make(DefaultLanguage)}
	for languageTag := range t.langStrings {
		if languageTag != DefaultLanguage && languageTag != "" {
			languageTags = append(languageTags, language.Raw.Make(languageTag))
		}
	}
	locale, _ := jibber_jabber.DetectIETF()
	match, _, _ := language.NewMatcher(languageTags).Match(language.Make(locale))
	return match.String()
}

// expandVariables takes a string with template variables like {{ var }} and
// expands them with the given map.
//
// Go templates are a bit different from Jinja templates, so this function
// wraps Go templates and allows one to write in a more Jinja-like
// {{var}} style instead of {{.var}} as the text/template package would
// demand.
func (t *Translator) expandVariables(str string, variables StringMap) (expanded string) {
	templ := template.New("")
	// Go template variables would have to be {{.var}}, but we want to use
	// {{var}}, so we simply define them as template function names returning
	// their string value.
	// That makes it slightly hacky here, but simplifies the translation
	// string files by not having to explain the leading dot on every
	// variable.
	funcMap := make(map[string]interface{})
	for key, value := range variables {
		// copy 'value', so it's is not the same string ref in all template functions
		boundValue := value[:]
		funcMap[key] = func() string { return boundValue }
	}
	templ, err := templ.Funcs(funcMap).Parse(str)
	if err != nil {
		log.Println(fmt.Sprintf("Invalid string template: '%s'", err))
		return str
	}
	var buf bytes.Buffer
	err = templ.Execute(&buf, nil)
	if err != nil {
		log.Println(fmt.Sprintf("Error executing template: '%s'", err))
		return str
	}
	return buf.String()
}
