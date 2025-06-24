package downloader

import (
	"os/exec"
)

// HasTarInPath checks if the 'tar' executable is available in the user's PATH.
func HasTarInPath() bool {
	_, err := exec.LookPath("tar")
	return err == nil
}
