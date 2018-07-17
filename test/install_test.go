package main

import (
	"testing"

	"github.com/grandchild/linux_installer"
)

func TestSize(t *testing.T) {
	i := linux_installer.install.InstallerNew()
	if i.Size() != 0 {
		t.Error("Size not zero")
	}
}
