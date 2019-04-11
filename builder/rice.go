// +build ignore
package linux_installer

// This whole file is not actually used as go code, it's just scanned by rice in the
// append process, when it's looking for directories from which to append data.
import "github.com/GeertJohan/go.rice"

func boxes() {
	// Modify/add/remove these lines to include different/more/fewer directories.
	// If you change, add or remove directories here, have them changed in
	// the installer base project's resources.go file as well!
	rice.FindBox("resources")
	rice.FindBox("data_compressed")
}
