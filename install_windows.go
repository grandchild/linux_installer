// +build windows

package linux_installer

// this code is untested!!

import (
	"golang.org/x/sys/windows"
	"path/filepath"
	"syscall"
	"unsafe"
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
