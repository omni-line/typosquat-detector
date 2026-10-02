//go:build !unix

package manifest

import "os"

func openNoFollow(path string) (*os.File, error) {
	return os.Open(path)
}
