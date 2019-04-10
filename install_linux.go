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
	desktopFileUserDir      = ".local/share/applications"
	desktopFileSystemDir    = "/usr/share/applications"
	desktopFilenameTemplate = `{{.company_short | lower | replace " " ""}}-{{.product | lower | replace " " ""}}.desktop`
	desktopFileTemplate     = `[Desktop Entry]
Name={{.product}}
Version={{.version}}
Type=Application
Icon={{.installDir}}/{{.iconFile}}
Exec={{.installDir}}/{{.startCommand}}
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

func osCreateLauncherEntry(variables VariableMap) (desktopFilepath string, err error) {
	content := ExpandVariables(desktopFileTemplate, variables)
	desktopFilename := ExpandVariables(desktopFilenameTemplate, variables)
	usr, err := user.Current()
	if err != nil {
		return
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

func osCreateUninstaller(installedFiles []string, variables VariableMap) error {
	uninstallScriptFilepath := filepath.Join(
		variables["installDir"], variables["uninstaller_name"]+".sh",
	)
	uninstallScriptTemplate, err := GetResource("uninstaller/uninstall.sh.template")
	if err != nil {
		return err
	}
	installedFiles = append(installedFiles, uninstallScriptFilepath)
	content := ExpandAllVariables(
		uninstallScriptTemplate,
		variables,
		UntypedVariableMap{"installedFiles": installedFiles},
	)
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

func osShowRawErrorDialog(message string) (err error) {
	_, err = exec.Command(
		"zenity",
		"--error",
		"--title", "error",
		"--no-wrap",
		"--text", message,
	).Output()
	return
}
