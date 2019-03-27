package main

import (
	"testing"

	"github.com/grandchild/linux_installer"
)

func TestNewInstallerSizeIsZero(t *testing.T) {
	i := linux_installer.install.NewInstaller()
	if i.Size() != 0 {
		t.Error("Size not zero")
	}
}
