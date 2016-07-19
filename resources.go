package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/GeertJohan/go.rice"
)

var installerBox *rice.Box
var dataBox *rice.Box

// Opens all payload boxes needed.
// For go.rice's 'append' mode to work, all calls to FindBox() have to be with
// a literal string parameter.
func openBoxes() {
	var err error
	installerBox, err = rice.FindBox("installer")
	if err != nil {
		panic(err)
	}
	dataBox, err = rice.FindBox("data")
	if err != nil {
		panic(err)
	}
}

func getInstallerResource(name string) (string, error) {
	return getResourceFiltered(installerBox, name, regexp.MustCompile(`.*`))
}
func getInstallerResourceFiltered(name string, dirFilter *regexp.Regexp) (string, error) {
	return getResourceFiltered(installerBox, name, dirFilter)
}
func getDataResource(name string) (string, error) {
	return getResourceFiltered(dataBox, name, regexp.MustCompile(`.*`))
}
func getDataResourceFiltered(name string, dirFilter *regexp.Regexp) (string, error) {
	return getResourceFiltered(dataBox, name, dirFilter)
}
func getResourceFiltered(box *rice.Box, name string, dirFilter *regexp.Regexp) (string, error) {
	if box == nil {
		return "", errors.New(fmt.Sprintf("Payload '%s' doesn't exist.", box))
	}
	text, err := box.String(name)
	if err != nil {
		contents := []string{}
		err = box.Walk(name, func(path string, info os.FileInfo, err error) error {
			if path != name {
				if dirFilter.FindStringIndex(path) != nil {
					contents = append(contents, path)
				}
				if info.IsDir() {
					return filepath.SkipDir
				}
			}
			return nil
		})
		if err == nil {
			text = strings.Join(contents, "\n")
		}
	}
	if err != nil {
		return "", errors.New(fmt.Sprint(name, " not found."))
	}
	return text, err
}
