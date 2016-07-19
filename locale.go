package main

import (
	"regexp"
	"strings"

	"github.com/cloudfoundry/jibber_jabber"
	"golang.org/x/text/language"
)

func getLocaleMatch() string {
	stringsFiles, _ := getInstallerResourceFiltered("strings", regexp.MustCompile(`\.json$`))
	langCodes := strings.Split(regexp.MustCompile(`.*/([^/]+)\.json`).ReplaceAllString(stringsFiles, "$1"), "\n")
	langTags := []language.Tag{language.Raw.Make("en")}
	for _, lang := range langCodes {
		if lang != "en" && lang != "" {
			langTags = append(langTags, language.Raw.Make(lang))
		}
	}
	locale, _ := jibber_jabber.DetectIETF()
	match, _, _ := language.NewMatcher(langTags).Match(language.Make(locale))
	return match.String()
}

func getAllLanguages() string {
	stringsFiles, _ := getInstallerResourceFiltered("strings", regexp.MustCompile(`\.json$`))
	langFiles := strings.Split(stringsFiles, "\n")
	langCodes := strings.Split(regexp.MustCompile(`.*/([^/]+)\.json`).ReplaceAllString(stringsFiles, "$1"), "\n")
	langs := []string{}
	for i, lang := range langCodes {
		if lang != "" {
			langStrings, _ := getInstallerResource(langFiles[i])
			langs = append(langs, "\""+lang+"\": "+langStrings)
		}
	}
	return "{\n" + strings.Join(langs, ",\n") + "\n}"
}
