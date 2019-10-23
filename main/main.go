package main

import (
	// this is the installer package name - here it refers to the parent directory
	"github.com/grandchild/linux_installer"
	"os"
)

func main() {
	os.Exit(linux_installer.Run())
}
