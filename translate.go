package linux_installer

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"sort"

	"github.com/cloudfoundry/jibber_jabber"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v2"
)

const (
	DefaultLanguage string = "en"
	displayKey             = "_language_display"
)

type Translator struct {
	language    string
	langStrings map[string]StringMap
	variables   StringMap
}

// NewTranslator returns a Translator without any variable lookup.
func NewTranslator() *Translator {
	return NewTranslatorVar(StringMap{})
}

// NewTranslatorVar returns a Translator with a variable lookup. It scans for any yaml
// files inside the languages folder in the resources box.
func NewTranslatorVar(variables StringMap) *Translator {
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
			return nil
		}
	}
	return &t
}

// Get returns the localized string for a given string key.
//
// The strings may contain template references to variables, which in turn may contain
// template references back to message strings. Only one round-trip of string ->
// variable -> string lookup is performed (i.e. a template variable in a localized
// string which is used by another template variable will not be expanded and the raw
// template would appear in the output.)
func (t *Translator) Get(key string) string {
	str := t.getRaw(key, t.language)
	return t.Expand(str)
}

// GetLanguage returns the identifier (e.g. "en") for the current language.
func (t *Translator) GetLanguage() string { return t.language }

// GetLanguages returns a list of identifiers for all available languages. The default
// language (if it has strings available) will be the first in the list, the rest is
// sorted alphabetically.
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
		languages = append([]string{DefaultLanguage}, languages...)
	}
	return languages
}

// GetAllStrings returns a string map of all strings for the current language, with
// variable templates expanded.
func (t *Translator) GetAllStrings() StringMap {
	strs := make(StringMap)
	for key := range t.langStrings[t.language] {
		strs[key] = t.Get(key)
	}
	return strs
}

// GetAllStringsRaw returns the unexpanded string map of all strings for the current
// language.
func (t *Translator) GetAllStringsRaw() StringMap { return t.langStrings[t.language] }

// GetAll returns a map of all localizations for a given string, indexed by the language
// code.
func (t *Translator) GetAll(key string) StringMap {
	versions := make(StringMap)
	for _, lang := range t.GetLanguages() {
		if value, ok := t.langStrings[lang][key]; ok {
			versions[lang] = t.expand(value, lang)
		} else {
			versions[lang] = ""
		}
	}
	return versions
}

// GetAllList returns a flat string list of all localizations for a given string key.
func (t *Translator) GetAllList(key string) (versions []string) {
	for _, lang := range t.GetLanguages() {
		if value, ok := t.langStrings[lang][key]; ok {
			versions = append(versions, t.expand(value, lang))
		}
	}
	return versions
}

// SetLanguage given a language code string (e.g.: "en"), sets the translator's
// language.
func (t *Translator) SetLanguage(language string) (err error) {
	if _, ok := t.langStrings[language]; !ok {
		return errors.New(fmt.Sprintf("No language '%s'.", language))
	}
	t.language = language
	return
}

// getLocale() returns the current system locale, as a language code string (e.g.:
// "en").
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

// Expand expands template variables in the given str (if any) with the translator's
// current language's strings.
func (t *Translator) Expand(str string) (expanded string) { return t.expand(str, t.language) }

// expand expands template variables in the given str (if any) with the translator's
// strings for the given language. If the language is not available in the translator,
// then an empty string is returned.
func (t *Translator) expand(str, language string) (expanded string) {
	availableLanguage := language
	if _, ok := t.langStrings[language]; !ok {
		availableLanguage = DefaultLanguage
	}
	if _, ok := t.langStrings[DefaultLanguage]; !ok {
		return ""
	}
	variables := make(map[string]string)
	for key, value := range t.variables {
		variables[key] = ExpandVariables(value, t.langStrings[availableLanguage])
	}
	return ExpandVariables(str, variables)
}

// getRaw returns a localized string for a given string key in a given language, without
// template expansion. If the language doesn't have strings available, then the default
// language is tried. If that fails as well, an empty string is returned.
func (t *Translator) getRaw(key, language string) string {
	if langStrings, ok := t.langStrings[language]; ok {
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
