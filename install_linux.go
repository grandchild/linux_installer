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
Icon={{.installDir}}/{{.iconFile}}
Exec={{.installDir}}/{{.startCommand}}
Comment={{.tagline}}
Categories=Simulation;Engineering;Science;
Terminal=true
`

	uninstallScriptFilename = "uninstall.sh"
	uninstallScriptTemplate = `#!/usr/bin/sh
uninstallFiles=(
    "$(dirname "$(readlink -f "$0")")"
    "{{.desktopFilepath}}"
)
echo "{{.uninstall_before}}: ${uninstallFiles[@]}"
echo -n "{{.uninstall_question}} "
read choice
if [ "${choice:0:1}" != "n" ] ; then
    rm -rf ${uninstallFiles[@]}
    if [ "$?" == "0" ] ; then
    	echo "{{.uninstall_success}}"
    else
    	echo "{{.uninstall_failure}}"
    fi
fi
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

func osCreateLauncherEntry(variables ...StringMap) (desktopFilepath string, err error) {
	content := ExpandVariables(desktopFileTemplate, MergeVariables(variables...))
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	var applicationsDir string
	if usr.Uid == "0" {
		applicationsDir = desktopFileSystemDir
	} else {
		applicationsDir = desktopFileUserDir
	}
	desktopFilepath = filepath.Join(usr.HomeDir, applicationsDir, desktopFilename)
	err = ioutil.WriteFile(desktopFilepath, []byte(content), 0755)
	return
}

func osCreateUninstaller(installDir string, variables ...StringMap) error {
	content := ExpandVariables(uninstallScriptTemplate, MergeVariables(variables...))
	uninstallScriptFilepath := filepath.Join(installDir, uninstallScriptFilename)
	return ioutil.WriteFile(uninstallScriptFilepath, []byte(content), 0755)
}
