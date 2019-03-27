// +build linux

package linux_installer

import (
	"io/ioutil"
	"os/user"
	"path/filepath"

	"golang.org/x/sys/unix"
)

const (
	desktopFileUserDir   = ".local/share/applications"
	desktopFileSystemDir = "/usr/share/applications"
	desktopFilename      = "acme-exampleapp.desktop"
	desktopFileTemplate  = `[Desktop Entry]
Name={{.product}}
Version={{.version}}
Type=Application
Icon={{.installDir}}/ExampleApp.png
Exec={{.installDir}}/ExampleApp
Comment={{.tagline}}
Categories=Simulation;Engineering;Science;
Terminal=true
`
)

func osFileWriteAccess(path string) bool {
	return unix.Access(path, unix.W_OK) == nil
}

func osDiskSpace(path string) int64 {
	fs := unix.Statfs_t{}
	if err := unix.Statfs(path, &fs); err != nil {
		return -1
	}
	return int64(fs.Bavail) * fs.Bsize
}

func osCreateLauncherEntry(variables ...StringMap) error {
	content := ExpandVariables(desktopFileTemplate, MergeVariables(variables...))
	usr, err := user.Current()
	if err != nil {
		return err
	}
	var applicationsDir string
	if usr.Uid == 0 {
		applicationsDir = desktopFileSystemDir
	} else {
		applicationsDir = desktopFileUserDir
	}
	desktopFilepath := filepath.Join(usr.HomeDir, applicationsDir, desktopFilename)
	return ioutil.WriteFile(desktopFilepath, []byte(content), 0755)
}
