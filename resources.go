package linux_installer

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"

	"github.com/GeertJohan/go.rice"
)

type BoxFile struct {
	path string
	info os.FileInfo
}

var resourcesBox *rice.Box
var dataBox *rice.Box

// openBoxes opens all payload boxes.
//
// For go.rice's 'append' mode to work, all calls to FindBox() have to have a literal
// string parameter. If you update directory names here, update builder/rice.go as well!
func openBoxes() {
	var err error
	resourcesBox, err = rice.FindBox("resources")
	if err != nil {
		panic(err)
	}
	dataBox, err = rice.FindBox("data_compressed")
	if err != nil {
		panic(err)
	}
}

// MustGetResource returns the contents of a resources file with the given name as a
// string. If the file does not exists it panics.
func MustGetResource(name string) string {
	content, err := GetResource(name)
	if err != nil {
		panic(err)
	}
	return content
}

// MustGetResourceFiltered returns the contents of multiple resource files within the
// subdir specified by name, and the filename regexp given by dirFilter. If the
// directory name does not exist it panics.
func MustGetResourceFiltered(name string, dirFilter *regexp.Regexp) map[string]string {
	resources, err := GetResourceFiltered(name, dirFilter)
	if err != nil {
		panic(err)
	}
	return resources
}

// GetResource returns the contents of of a resources file with the given name as a
// string. If the file does not exists it returns an error.
func GetResource(name string) (string, error) { return getBoxContent(resourcesBox, name) }

// GetResourceFiltered returns the contents of multiple resource files within the subdir
// specified by name, and the filename regexp given by dirFilter. If the directory name
// does not exist it returns an error.
func GetResourceFiltered(name string, dirFilter *regexp.Regexp) (map[string]string, error) {
	return getBoxContentFiltered(resourcesBox, name, dirFilter)
}

// UnpackResourceDir copies all resource files from a subdir given by from to a path
// given by to. It returns an error if the boxes aren't opened yet, the path can't be
// written to, or anything else goes wrong.
func UnpackResourceDir(from string, to string) error { return unpackDir(resourcesBox, from, to) }

// UnpackDataDir copies all data files from a subdir given by from to a path given by
// to. It returns an error if the boxes aren't opened yet, the path can't be written to,
// or anything else goes wrong.
func UnpackDataDir(from string, to string) error { return unpackDir(dataBox, from, to) }

// getBoxContent returns the content of a file given by name inside a given rice box.
func getBoxContent(box *rice.Box, name string) (string, error) {
	if box == nil {
		return "", errors.New("Boxes not opened yet.")
	}
	return box.String(name)
}

// getBoxContentFiltered returns the content of multiple files inside a given rice box.
// It returns contents for all files within the directory given by name and whose
// filenames match the dirFilter regexp.
func getBoxContentFiltered(box *rice.Box, name string, dirFilter *regexp.Regexp) (map[string]string, error) {
	contents := make(map[string]string)
	if box == nil {
		return contents, errors.New("Boxes not opened yet.")
	}
	err := box.Walk(name, func(path string, info os.FileInfo, err error) error {
		if path != name {
			if dirFilter.FindStringIndex(path) != nil {
				content, err := box.String(path)
				if err == nil {
					contents[path] = content
				}
			}
			if info.IsDir() {
				return filepath.SkipDir
			}
		}
		return nil
	})
	if err != nil {
		return contents, errors.New(fmt.Sprint(name, " not found."))
	}
	return contents, err
}

// unpackFile copies a single file from a given rice box, from fromPath inside the box
// to toPath on the filesystem.
func unpackFile(box *rice.Box, fromPath string, toPath string) error {
	var err error
	if box == nil {
		return errors.New("Boxes not opened yet.")
	}
	from, err := box.Open(fromPath)
	if err != nil {
		return err
	}
	defer from.Close()
	err = os.MkdirAll(path.Dir(toPath), 0755)
	if err != nil {
		log.Println(fmt.Sprintf("%s %s", err, toPath))
		return err
	}
	to, err := os.OpenFile(toPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer to.Close()
	_, err = io.Copy(to, from)
	if err != nil && err.Error() != "EOF" {
		return err
	}
	return nil
}

// unpackDir copies a directory, recursively, from a given rice box, from fromPath
// inside the box to toPath on the filesystem.
func unpackDir(box *rice.Box, fromPath string, toPath string) error {
	if box == nil {
		return errors.New("Boxes not opened yet.")
	}
	err := os.MkdirAll(toPath, 0755)
	if err != nil {
		log.Println(fmt.Sprintf("%s %s", err, toPath))
		return err
	}
	err = box.Walk(fromPath, func(path string, info os.FileInfo, err error) error {
		relPath, _ := filepath.Rel(fromPath, path)
		outPath := filepath.Join(toPath, relPath)
		if info.IsDir() {
			err = os.MkdirAll(outPath, 0755)
		} else {
			err = unpackFile(box, path, outPath)
		}
		return err
	})
	return err
}
