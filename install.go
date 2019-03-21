package linux_installer

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

type (
	// InstallFile is an augmented os.FileInfo struct with both source and
	// target path as well as a flag indicating wether the file has been
	// copied to the target or not.
	InstallFile struct {
		*zip.File
		Target    string
		installed bool
	}
	// InstallStatus is a message struct that gets passed around at various
	// times in the installation process. All fields are optional and contain
	// the current file, wether the installer as a whole is finished or not,
	// or wether it's been aborted and rolled back.
	InstallStatus struct {
		File    *InstallFile
		Done    bool
		Aborted bool
	}
	// Installer represents a set of files and a target to be copied into. It
	// contains information about the files, size, and status (done or not),
	// as well as 3 different message channels, for each abort and its
	// confirmation as well as status channel.
	Installer struct {
		Target              string
		Done                bool
		tempPath            string
		totalSize           int64
		installedSize       int64
		files               []*InstallFile
		statusChannel       chan InstallStatus
		abortChannel        chan bool
		abortConfirmChannel chan bool
		actionLock          sync.Mutex
		progressFunction    func(InstallStatus)
	}
)

// InstallerNew creates a new Installer. You will still need to set the target
// path after initialization:
//
// 	installer := InstallerNew()
// 	/* ... some other stuff happens ... */
// 	installer.Target = "/some/output/path"
// 	/* and go: */
// 	installer.StartInstall()
//
// Alternatively you can just use InstallerToNew() and set the target
// directly:
//
// 	installer := InstallerToNew("/some/output/path/")
// 	installer.StartInstall()
// 	/* some watch loop with 'installer.Status()' */
//
func InstallerNew(tempPath string) Installer { return InstallerToNew("", tempPath) }

// InstallerToNew creates a new installer with a target path.
func InstallerToNew(target string, tempPath string) Installer {
	return Installer{
		Target:              target,
		tempPath:            tempPath,
		totalSize:           DataSize(), // FIXME report size of zip contents
		statusChannel:       make(chan InstallStatus, 1),
		abortChannel:        make(chan bool, 1),
		abortConfirmChannel: make(chan bool, 1),
		progressFunction:    func(status InstallStatus) {},
	}
}

// StartInstall runs the installer in a separate goroutine and returns
// immediately. Use Status() to get updates about the progress.
func (i *Installer) StartInstall() { go i.install() }

// StartInstallFromSubdir is the same as StartInstall but only installs a
// subset of the source data.
func (i *Installer) StartInstallFromSubdir(subdir string) { go i.installFromSubdir(subdir) }

// install runs the installation.
func (i *Installer) install() error { return i.installFromSubdir("") }

// installFromSubdir runs the installation.
func (i *Installer) installFromSubdir(subdir string) error {
	i.Done = false
	i.actionLock.Lock()
	defer i.actionLock.Unlock()

	log.Printf("Unpacking temp data.zip")
	reader, err := i.unpackDataZip()
	if err != nil {
		return err
	}

	i.files = make([]*InstallFile, 0, len(reader.File))
	for _, file := range reader.File {
		if !strings.HasPrefix(file.Name, subdir) {
			continue
		}
		relPath, err := filepath.Rel(subdir, file.Name)
		if err != nil {
			continue
		}
		// Check for ZipSlip vulnerability and ignore any files with invalid
		// paths. See: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(
			filepath.Join(i.Target, relPath),
			filepath.Clean(i.Target)+string(os.PathSeparator),
		) {
			continue
		}
		i.files = append(
			i.files,
			&InstallFile{file, filepath.Join(i.Target, relPath), false},
		)
	}
	for _, file := range i.files {
		select {
		case <-i.abortChannel:
			i.Done = false
			i.abortConfirmChannel <- true
			return err
		default:
			// log.Printf("Installing file/dir %s", file.Target)
			status := InstallStatus{File: file}
			i.setStatus(status)
			i.progressFunction(status)
			if file.FileInfo().IsDir() {
				os.MkdirAll(file.Target, 0755)
			} else {
				os.MkdirAll(filepath.Dir(file.Target), 0755)
				err = installFile(file)
				if err != nil {
					return err
				}
				i.installedSize += int64(file.UncompressedSize64)
			}
			file.installed = true
			i.setStatus(InstallStatus{File: file})
		}
	}
	i.Done = true
	i.setStatus(InstallStatus{Done: true})
	return err
}

// UnpackDataZip extracts the appended data zipfile to the temporary directory
// given by tempPath.
func (i *Installer) unpackDataZip() (*zip.ReadCloser, error) {
	dataTempFilepath := filepath.Join(i.tempPath, "data", "data.zip")
	i.setStatus(InstallStatus{File: &InstallFile{Target: dataTempFilepath}})
	err := UnpackDataFile("data.zip", dataTempFilepath)
	if err != nil {
		return nil, err
	}
	return zip.OpenReader(dataTempFilepath)
}

func installFile(file *InstallFile) error {
	targetFile, err := os.OpenFile(
		file.Target,
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		file.Mode()|0600, // use file's permissions, but make sure that user has at least rw (=0600)
	)
	if err != nil {
		return err
	}
	fileReader, err := file.Open()
	if err != nil {
		return err
	}
	_, err = io.Copy(targetFile, fileReader)
	targetFile.Close()
	fileReader.Close()
	if err != nil {
		return err
	}
	err = os.Chtimes(file.Target, time.Now(), file.Modified)
	return err
}

// Abort can be called to stop the installer. The installer will usually not
// stop immediately, but finish copying the current file.
//
// Use Rollback() instead of Abort() if you also want all files and directories
// rolled back and deleted.
func (i *Installer) Abort() {
	i.abortChannel <- true
	<-i.abortConfirmChannel
}

// Rollback can be used to abort and roll back (i.e. delete) the files and
// directories that have been installed so far. It will not delete files that
// haven't been written by the installer, but will delete any file that was
// overwritten by it.
//
// Rollback implicitly calls Abort().
func (i *Installer) Rollback() {
	i.Abort()
	i.actionLock.Lock()
	defer i.actionLock.Unlock()
	// Do not os.RemoveAll(i.Target)! That could easily delete files and
	// folders not created by the installer.
	for p := len(i.files) - 1; p >= 0; p-- {
		if i.files[p].installed {
			// log.Printf("Rolling back: %s", i.files[p].Target)
			err := os.Remove(i.files[p].Target)
			if err != nil {
				log.Printf("Error deleting %s", i.files[p].Target)
			}
			i.files[p].installed = false
			if !i.files[p].FileInfo().IsDir() {
				i.installedSize -= int64(i.files[p].UncompressedSize64)
			}
			i.setStatus(InstallStatus{File: i.files[p]})
		}
	}
	os.RemoveAll(filepath.Join(i.tempPath, "data"))
	i.Done = true
	i.setStatus(InstallStatus{Aborted: true})
}

// setStatus is a non-blocking write to the status channel. If no-one is
// listening through Status() then it will simply do nothing and return.
func (i *Installer) setStatus(status InstallStatus) {
	select {
	case i.statusChannel <- status:
	case <-time.After(1 * time.Second):
	}
}

// Status returns the current installer status as an InstallerStatus object.
func (i *Installer) Status() InstallStatus {
	select {
	case status := <-i.statusChannel:
		return status
	case <-time.After(1 * time.Second):
		return InstallStatus{}
	}
}

// CheckInstallDir checks if the given directory is a valid path, creating it
// if it doesn't exist.
func (i *Installer) CheckInstallDir(dirName string) error {
	parent := path.Dir(dirName)
	parentInfo, err := os.Stat(parent)
	// log.Println(fmt.Sprintf("Checking install location: '%s'", dirName))
	if err != nil || !parentInfo.IsDir() {
		return errors.New(fmt.Sprintf("Install parent is not dir: '%s'", parent))
	} else if unix.Access(parent, unix.W_OK) != nil {
		return errors.New(fmt.Sprintf("Install location is not writeable: '%s' -> '%s'", parent, parentInfo.Mode().Perm()))
	}
	i.Target = dirName
	return nil
}

// NextFile returns the file that the installer will install next, or the one
// that is currently being installed.
func (i *Installer) NextFile() *InstallFile {
	for _, file := range i.files {
		if !file.installed {
			return file
		}
	}
	return nil
}

func (i *Installer) SetProgressFunction(function func(InstallStatus)) {
	i.progressFunction = function
}

// Progress returns the size ratio between already installed files and all
// files. The result is a float between 0.0 and 1.0, inclusive.
func (i *Installer) Progress() float64 {
	return float64(i.installedSize) / float64(i.totalSize)
}

// Size returns the bytes that have been copied so far or should be copied in
// total.
func (i *Installer) Size() int64 {
	if i.Done {
		return i.totalSize
	} else {
		return i.installedSize
	}
}

// SizeString returns a human-readable version of Size(), appending a size
// suffix, as needed.
func (i *Installer) SizeString() string {
	size := i.Size()
	switch {
	case size < KB:
		return fmt.Sprintf("%dB", size)
	case size < MB:
		return fmt.Sprintf("%.2fKB", float64(size)/float64(KB))
	case size < GB:
		return fmt.Sprintf("%.2fMB", float64(size)/float64(MB))
	case size < TB:
		return fmt.Sprintf("%.2fGB", float64(size)/float64(GB))
	default:
		return fmt.Sprintf("%.2fTB", float64(size)/float64(TB))
	}
}

// WaitForDone returns only after the installer has finished installing (or
// rolling back).
func (i *Installer) WaitForDone() {
	for {
		if status := <-i.statusChannel; status.Done {
			return
		}
	}
}
