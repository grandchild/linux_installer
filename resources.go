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

var pBox *rice.Box

func openResource(name string) {
	var err error
	pBox, err = rice.FindBox(name)
	if err != nil {
		panic(err)
	}
}

func getResource(name string) (string, error) {
	return getResourceFiltered(name, regexp.MustCompile(`.*`))
}
func getResourceFiltered(name string, dirFilter *regexp.Regexp) (string, error) {
	if pBox == nil {
		return "", errors.New(fmt.Sprintf("payload '%s' doesn't exist.", pBox))
	}
	text, err := pBox.String(name)
	if err != nil {
		contents := []string{}
		err = pBox.Walk(name, func(path string, info os.FileInfo, err error) error {
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
