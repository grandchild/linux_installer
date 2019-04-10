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
)

const (
	B   int64 = 1
	KiB       = 1024 * B
	MiB       = 1024 * KiB
	GiB       = 1024 * MiB
	TiB       = 1024 * GiB
	PiB       = 1024 * TiB
)

type (
	// InstallFile is an augmented zip.FileInfo struct with both source and target path as
	// well as a flag indicating wether the file has been copied to the target or not.
	// Source and target path will be the same if the installation doesn't run from a
	// subdir of the source data.
	InstallFile struct {
		*zip.File
		Target    string
		installed bool
	}
	// InstallStatus is a message struct that gets passed around at various times in the
	// installation process. All fields are optional and contain the current file, a status
	// string, wether the installer as a whole is finished or not, or wether it's been
	// aborted and rolled back.
	InstallStatus struct {
		S       string
		File    *InstallFile
		Done    bool
		Aborted bool
	}
	// Installer represents a set of files and a target to be copied into. It contains
	// information about the files, size, and status (done or not), as well as 3 different
	// message channels, for each abort and its confirmation as well as status channel.
	Installer struct {
		Target               string
		Status               *InstallStatus
		CreateLauncher       bool
		Done                 bool
		tempPath             string
		dataPrepared         bool
		hooksPrepared        bool
		existingTargetParent string
		totalSize            int64
		installedSize        int64
		files                []*InstallFile
		doneChannel          chan bool
		abortChannel         chan bool
		abortConfirmChannel  chan bool
		actionLock           sync.Mutex
		progressFunction     func(InstallStatus)
		config               *Config
		err                  error
	}
)

// NewInstaller creates a new Installer. You will still need to set the target
// path after initialization:
//
// 	installer := NewInstaller()
// 	/* ... some other stuff happens ... */
// 	installer.Target = "/some/output/path"
// 	/* and go: */
// 	installer.StartInstall()
//
// Alternatively you can just use NewInstallerTo() and set the target
// directly:
//
// 	installer := NewInstallerTo("/some/output/path/")
// 	installer.StartInstall()
// 	/* some watch loop with 'installer.Status()' */
//
func NewInstaller(tempPath string, config *Config) Installer {
	return NewInstallerTo("", tempPath, config)
}

// NewInstallerTo creates a new installer with a target path.
func NewInstallerTo(target string, tempPath string, config *Config) Installer {
	return Installer{
		Target:              target,
		CreateLauncher:      true,
		Status:              &InstallStatus{},
		tempPath:            tempPath,
		doneChannel:         make(chan bool, 1),
		abortChannel:        make(chan bool, 1),
		abortConfirmChannel: make(chan bool, 1),
		progressFunction:    func(status InstallStatus) {},
		config:              config,
	}
}

// StartInstall runs the installer in a separate goroutine and returns immediately. Use
// Status() to get updates about the progress.
func (i *Installer) StartInstall() {
	go i.install()
}

// prepareDataFiles unpacks data.zip into the temp directory and scans the contents.
func (i *Installer) prepareDataFiles() error {
	if i.dataPrepared {
		return nil
	}
	err := UnpackDataDir("", filepath.Join(i.tempPath, "data"))
	if err != nil {
		return err
	}
	reader, err := zip.OpenReader(filepath.Join(i.tempPath, "data", i.config.DataFilename))
	if err != nil {
		return err
	}

	i.dataPrepared = false
	i.totalSize = 0
	i.files = make([]*InstallFile, 0, len(reader.File))
	for _, file := range reader.File {
		// Check for ZipSlip vulnerability and ignore any files with invalid paths.
		// See: https://snyk.io/research/zip-slip-vulnerability#go
		dummyTarget := "/some/dir/"
		if !strings.HasPrefix(
			filepath.Join(dummyTarget, file.Name),
			filepath.Clean(dummyTarget)+string(os.PathSeparator),
		) {
			continue
		}
		i.files = append(
			i.files,
			&InstallFile{file, file.Name, false},
		)
		i.totalSize += int64(file.UncompressedSize64)
	}
	i.dataPrepared = true
	return err
}

func (i *Installer) prepareHooks() error {
	if i.hooksPrepared {
		return nil
	}
	err := UnpackResourceDir("hooks", filepath.Join(i.tempPath, "hooks"))
	if err == nil {
		i.hooksPrepared = true
	}
	return err
}

// install runs the installation. It loops through all files collected by
// prepareDataFiles, creates directories as necessary and calls installFile on each
// file.
func (i *Installer) install() {
	i.Done = false
	i.Status = &InstallStatus{}
	i.actionLock.Lock()
	defer i.actionLock.Unlock()

	var err error
	if !i.dataPrepared {
		err = i.prepareDataFiles()
		if err != nil {
			i.err = err
			return
		}
	}

	os.MkdirAll(i.Target, 0755)
	for _, file := range i.files {
		select {
		case <-i.abortChannel:
			i.Done = false
			i.abortConfirmChannel <- true
			i.err = err
			return
		default:
			log.Printf("Installing file/dir %s", i.fileTarget(file))
			i.Status = &InstallStatus{S: file.Name, File: file}
			i.progressFunction(*i.Status)
			if file.FileInfo().IsDir() {
				os.MkdirAll(i.fileTarget(file), 0755)
			} else {
				os.MkdirAll(filepath.Dir(i.fileTarget(file)), 0755)
				err = i.installFile(file)
				if err != nil {
					i.err = err
					return
				}
				i.installedSize += int64(file.UncompressedSize64)
			}
			file.installed = true
			i.Status = &InstallStatus{File: file}
		}
	}
	os.RemoveAll(filepath.Join(i.tempPath, "data"))
	i.Done = true
	i.Status = &InstallStatus{Done: true}
	i.doneChannel <- true
	i.err = err
}

// installFile copies a file into the target location.
//
// The file will have the same permissions as the source file, except for read and write
// permissions for the owning user, which are always given.
func (i *Installer) installFile(file *InstallFile) error {
	targetFile, err := os.OpenFile(
		i.fileTarget(file),
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		file.Mode()|0600, // user has at least read/write
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
	err = os.Chtimes(i.fileTarget(file), time.Now(), file.Modified)
	return err
}

// fileTarget returns the complete target path of a file, from the installer's Target
// path and the file's relative Target path.
func (i *Installer) fileTarget(file *InstallFile) string {
	return filepath.Join(i.Target, file.Target)
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
			err := os.Remove(i.fileTarget(i.files[p]))
			if err != nil {
				log.Printf("Error deleting %s\n", i.fileTarget(i.files[p]))
			} else {
				log.Printf("Rolled back: %s\n", i.fileTarget(i.files[p]))
			}
			i.files[p].installed = false
			if !i.files[p].FileInfo().IsDir() {
				i.installedSize -= int64(i.files[p].UncompressedSize64)
			}
			i.Status = &InstallStatus{File: i.files[p]}
		}
	}
	i.Done = true
	i.doneChannel <- true
	i.Status = &InstallStatus{Aborted: true}
}

// CheckInstallDir checks if the given directory is a valid path. Returns err when the
// dirName exists but is not a directory, or when dirName or the lowest existing parent
// is not writable.
func (i *Installer) CheckInstallDir(dirName string) error {
	parent := path.Dir(path.Clean(dirName))
	for parent != string(os.PathSeparator) || parent != "." {
		parentInfo, err := os.Stat(parent)
		if err != nil || !parentInfo.IsDir() {
			parent = path.Dir(parent)
		} else {
			break
		}
	}
	i.existingTargetParent = parent
	parentInfo, err := os.Stat(parent)
	if err != nil || !parentInfo.IsDir() {
		return errors.New("path_err_not_dir")
		// fmt.Sprintf("Install parent is not dir: '%s'", parent)
	} else if !osFileWriteAccess(parent) { // os-specific
		return errors.New("path_err_not_writable")
		// fmt.Sprintf("Install location is not writeable: '%s' -> '%s'", parent, parentInfo.Mode().Perm())
	}
	if err != nil {
		return errors.New("path_err_other")
	}
	i.Target = path.Clean(dirName)
	return nil
}

// NextFile returns the file that the installer will install next, or the one that is
// currently being installed.
func (i *Installer) NextFile() *InstallFile {
	for _, file := range i.files {
		if !file.installed {
			return file
		}
	}
	return nil
}

// SetProgressFunction takes a function which receives an InstallStatus, and sets it to
// be called whenever the function
func (i *Installer) SetProgressFunction(function func(InstallStatus)) {
	i.progressFunction = function
}

// Progress returns the size ratio between already installed files and all files. The
// result is a float between 0.0 and 1.0, inclusive.
func (i *Installer) Progress() float64 {
	if i.totalSize == 0 {
		return 0.0
	}
	return float64(i.installedSize) / float64(i.totalSize)
}

// diskSpace returns the user-available disk space in bytes
func (i *Installer) diskSpace() int64 {
	// os-specific
	return osDiskSpace(i.existingTargetParent)
}

// DiskSpaceSufficient returns true when the total size of files to be installed is
// smaller than the remaining available space on the disk that contains the installer's
// target path.
func (i *Installer) DiskSpaceSufficient() bool {
	return i.totalSize < i.diskSpace()
}

// SizeString returns a human-readable version of Size(), appending a size suffix as
// needed.
func (i *Installer) SizeString() string {
	i.prepareDataFiles()
	return i.sizeString(i.totalSize)
}
func (i *Installer) SpaceString() string { return i.sizeString(i.diskSpace()) }
func (i *Installer) sizeString(bytes int64) string {
	switch {
	case bytes < KiB:
		return fmt.Sprintf("%d B", bytes)
	case bytes < MiB:
		return fmt.Sprintf("%.2f KiB", float64(bytes)/float64(KiB))
	case bytes < GiB:
		return fmt.Sprintf("%.2f MiB", float64(bytes)/float64(MiB))
	case bytes < TiB:
		return fmt.Sprintf("%.2f GiB", float64(bytes)/float64(GiB))
	case bytes < PiB:
		return fmt.Sprintf("%.2f TiB", float64(bytes)/float64(TiB))
	default:
		return fmt.Sprintf("%.2f PiB", float64(bytes)/float64(PiB))
	}
}

// WaitForDone returns only after the installer has finished installing (or
// rolling back).
func (i *Installer) WaitForDone() {
	for {
		if done := <-i.doneChannel; done {
			return
		}
	}
}

// PreInstall runs
func (i *Installer) PreInstall() {
	i.prepareHooks()
	i.Status = &InstallStatus{S: "pre"}
	err := osRunHookIfExists(filepath.Join(i.tempPath, "hooks", "pre-install"))
	if err != nil {
		i.err = err
	}
}

// PostInstall runs a post-install script & creates an uninstaller as well as an
// optional launcher entry for the program.
func (i *Installer) PostInstall(variablesList ...VariableMap) {
	i.prepareHooks()
	i.Status = &InstallStatus{S: "post"}
	err := osRunHookIfExists(filepath.Join(i.tempPath, "hooks", "post-install"))
	if err != nil {
		i.err = err
		return
	}
	uninstallerFileList := make([]string, 0, len(i.files)+1) // +1 for launcher shortcut
	// reversed -> delete dir content before dir
	for j := len(i.files) - 1; j >= 0; j-- {
		if i.files[j].installed {
			uninstallerFileList = append(uninstallerFileList, i.fileTarget(i.files[j]))
		}
	}
	variablesList = append(variablesList, VariableMap{"installDir": i.Target})
	variables := MergeVariables(variablesList...)
	if i.CreateLauncher {
		launcherFile, err := osCreateLauncherEntry(variables)
		if err == nil {
			uninstallerFileList = append(uninstallerFileList, launcherFile)
			err = osCreateUninstaller(uninstallerFileList, variables)
			if err != nil {
				log.Println(err.Error())
			}
		}
	} else {
		err = osCreateUninstaller(uninstallerFileList, variables)
		if err != nil {
			log.Println(err.Error())
		}
	}
}
