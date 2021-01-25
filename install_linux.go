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
	"syscall"

	"golang.org/x/sys/unix"
)

const (
	desktopFileUserDir      = ".local/share/applications"
	desktopFileSystemDir    = "/usr/share/applications"
	desktopFilenameTemplate = `{{if .company_short}}{{.company_short | lower | replace " " ""}}-{{end}}{{.product | lower | replace " " ""}}.desktop`
	desktopFileTemplate     = `[Desktop Entry]
Name={{.product}}
Version={{.version}}
Type=Application
Icon={{.installDir}}/{{.icon_file}}
Exec={{.installDir}}/{{.start_command}}
Comment={{.tagline}}
Terminal={{.show_terminal_during_app_run}}
`
)

// osFileWriteAccess returns whether a given path has write access for the current user.
func osFileWriteAccess(path string) bool {
	return unix.Access(path, unix.W_OK) == nil
}

// osDiskSpace returns the amount of bytes available to the current user on the
// partition that the given path resides on.
func osDiskSpace(path string) int64 {
	fs := unix.Statfs_t{}
	if err := unix.Statfs(path, &fs); err != nil {
		return -1
	}
	return int64(fs.Bavail) * fs.Bsize
}

// osCreateLauncherEntry creates an application menu entry for the application being
// installed.
//
// On linux this creates a .desktop file in the users application dir, or—if
// installing as root—in the system-wide application dir.
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
	err = os.MkdirAll(filepath.Join(usr.HomeDir, applicationsDir), 0755)
	if err != nil {
		return
	}
	desktopFilepath = filepath.Join(usr.HomeDir, applicationsDir, desktopFilename)
	err = ioutil.WriteFile(desktopFilepath, []byte(content), 0755)
	return
}

// osCreateUninstaller expands the uninstaller template with installed files that were
// installed, and writes the result into a file that removes the installed application
// when executed.
//
// The uninstaller script template(s) are located in the resources/uninstaller/
// directory.
//
// On Linux, this is a simple .sh script with a list of files and directories to be fed
// to rm and rmdir respectively.
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

// osRunHookIfExists runs a script given its base name (no extension), if that script
// does exist. The hook scripts are located in the resources/hooks/ directory.
// installPath is the installation directory, and the script can expect it as its first
// commandline argument.
//
// On Linux it loads the hook files that end in ".sh".
func osRunHookIfExists(scriptFile string, installPath string) error {
	if _, err := os.Stat(scriptFile + ".sh"); os.IsNotExist(err) {
		return nil
	}
	err := os.Chmod(scriptFile+".sh", 0755)
	out, err := exec.Command("/bin/sh", scriptFile+".sh", installPath).CombinedOutput()
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

// osShowRawErrorDialog tries to show a graphical error dialog in case the main GUI
// fails to load. If that fails too, an error is returned.
//
// On Linux it tries to run a Zenity command.
func osShowRawErrorDialog(message string) (err error) {
	_, err = exec.Command(
		"zenity",
		"--error",
		"--title", "error",
		"--no-wrap",
		"--text", message,
	).CombinedOutput()
	return
}

// osExecVE runs cmd with the given args, replaces the current process and never
// returns.
func osExecVE(cmd string, args []string) {
	err := syscall.Exec(cmd, args, os.Environ())
	if err != nil {
		log.Println("execve error:", err.Error())
		os.Exit(1)
	}
}
