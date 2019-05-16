package main

import (
	"testing"

	"github.com/grandchild/linux_installer"
)

func TestNewInstallerSizeIsZero(t *testing.T) {
	i := installer.NewInstaller("", &installer.Config{})
	if i.SizeString() != "0 B" {
		t.Error("Size not zero")
	}
}
