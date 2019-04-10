// +build ignore
package linux_installer

// This whole file is not actually used as go code, it's just scanned by rice in the
// append process, when it's looking for directories from which to append data.
import "github.com/GeertJohan/go.rice"

func boxes() {
	// Modify/add/remove these lines to include different/more/fewer directories.
	rice.FindBox("resources")
	rice.FindBox("data_compressed")
}
