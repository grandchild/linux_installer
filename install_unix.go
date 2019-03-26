// +build linux darwin

package linux_installer

import "golang.org/x/sys/unix"

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
