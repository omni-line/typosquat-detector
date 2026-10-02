//go:build unix

package manifest

import (
	"os"
	"syscall"
)

// openNoFollow opens path without following a final symlink and without
// blocking if the file was swapped for a FIFO after the Lstat check.
func openNoFollow(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
}
