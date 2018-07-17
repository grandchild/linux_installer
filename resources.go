package linux_installer

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	// "strings"

	"github.com/GeertJohan/go.rice"
	// "github.com/grandchild/go.rice"
)

type BoxFile struct {
	path string
	info os.FileInfo
}

const (
	B  int64 = 1
	KB       = 1024 * B
	MB       = 1024 * KB
	GB       = 1024 * MB
	TB       = 1024 * GB
)

var resourcesBox *rice.Box
var dataBox *rice.Box

// Opens all payload boxes needed.
// For go.rice's 'append' mode to work, all calls to FindBox() have to be with
// a literal string parameter.
func openBoxes() {
	var err error
	resourcesBox, err = rice.FindBox("resources")
	if err != nil {
		panic(err)
	}
	dataBox, err = rice.FindBox("data")
	if err != nil {
		panic(err)
	}
}

func DataSize() int64 {
	return boxSize(dataBox)
}

func MustGetResource(name string) string {
	content, err := GetResource(name)
	if err != nil {
		panic(err)
	}
	return content
}
func MustGetResourceFiltered(name string, dirFilter *regexp.Regexp) map[string]string {
	resources, err := GetResourceFiltered(name, dirFilter)
	if err != nil {
		panic(err)
	}
	return resources
}

func GetResource(name string) (string, error) { return getBoxContent(resourcesBox, name) }
func GetResourceFiltered(name string, dirFilter *regexp.Regexp) (map[string]string, error) {
	return getBoxContentFiltered(resourcesBox, name, dirFilter)
}
func GetData(name string) (string, error) { return getBoxContent(dataBox, name) }
func GetDataFiltered(name string, dirFilter *regexp.Regexp) (map[string]string, error) {
	return getBoxContentFiltered(dataBox, name, dirFilter)
}
func ListDataDir(name string) ([]BoxFile, error)      { return listDir(dataBox, name) }
func UnpackResourceFile(from string, to string) error { return unpackFile(resourcesBox, from, to) }
func UnpackDataFile(from string, to string) error     { return unpackFile(dataBox, from, to) }
func UnpackResourceDir(from string, to string) error  { return unpackDir(resourcesBox, from, to) }
func UnpackDataDir(from string, to string) error      { return unpackDir(dataBox, from, to) }

func getBoxContent(box *rice.Box, name string) (string, error) {
	if box == nil {
		return "", errors.New("Boxes not opened yet.")
	}
	return box.String(name)
}
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

func listDir(box *rice.Box, name string) ([]BoxFile, error) {
	list := []BoxFile{}
	if box == nil {
		return list, errors.New("Boxes not opened yet.")
	}
	err := box.Walk(name, func(path string, info os.FileInfo, err error) error {
		list = append(list, BoxFile{path: path, info: info})
		return err
	})
	return list, err
}

func boxSize(box *rice.Box) int64 {
	if box == nil {
		return 0
	}
	var size int64 = 0
	err := box.Walk("", func(_ string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	if err != nil {
		return 0
	}
	return size
}
