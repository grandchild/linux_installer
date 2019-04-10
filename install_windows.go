// +build windows

package linux_installer

// This code is unused & untested!! (And probably completely unnecessary...)

import (
	"errors"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

func osFileWriteAccess(path string) (success bool) {
	testPath := syscall.StringToUTF16Ptr(filepath.Join(path, ".test"))
	_, err := windows.CreateFile(
		testPath,
		windows.GENERIC_WRITE|windows.GENERIC_READ,
		0,
		nil,
		windows.CREATE_NEW,
		windows.FILE_ATTRIBUTE_HIDDEN,
		0,
	)
	if err == nil {
		windows.DeleteFile(testPath)
		return true
	} else {
		return false
	}
}

func osDiskSpace(path string) (availableBytes int64) {
	win32 := syscall.MustLoadDLL("kernel32.dll")
	getDiskFreeSpace := win32.MustFindProc("GetDiskFreeSpaceExW")
	getDiskFreeSpace.Call(
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(path))),
		uintptr(0),
		uintptr(0),
		uintptr(unsafe.Pointer(&availableBytes)),
	)
	return
}

func osCreateLauncherEntry(variables VariableMap) (desktopFilepath string, err error) {
	return
}
func osCreateUninstaller(uninstallerFileList []string, variables VariableMap) error {
	return nil
}

func osRunHookIfExists(scriptFile string) (err error) {
	if _, err = os.Stat(scriptFile + ".bat"); os.IsNotExist(err) {
		return
	}
	out, err := exec.Command(scriptFile + ".bat").Output()
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

func osShowRawErrorDialog(message string) (err error) { return }
