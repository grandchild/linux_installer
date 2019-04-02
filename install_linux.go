// +build linux

package linux_installer

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
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
    {{ if .desktopFilepath }}"{{.desktopFilepath}}"{{ end }}
)
echo "{{.uninstall_before}}: ${uninstallFiles[@]}"
echo -n '{{.uninstall_question}} '
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

func osRunHookIfExists(scriptFile string) error {
	if _, err := os.Stat(scriptFile + ".sh"); os.IsNotExist(err) {
		return nil
	}
	out, err := exec.Command("/bin/sh", scriptFile+".sh").Output()
	log.Println("hook output:\n", string(out[:]))
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return errors.New(string(exitErr.Stderr))
		} else {
			return err
		}
	}
	return err
}
