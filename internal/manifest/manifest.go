// Package manifest holds shared dependency types and safe file reads.
package manifest

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
)

// Dependency is a package declared in a manifest.
type Dependency struct {
	Name    string
	Version string
	Group   string
	Line    int
}

const MaxFileSize = 10 << 20

var ErrTooLarge = fmt.Errorf("manifest exceeds %d MiB limit", MaxFileSize>>20)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// ReadFile reads a regular file capped at MaxFileSize and strips a UTF-8 BOM.
func ReadFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !fi.Mode().IsRegular() {
		return nil, errors.New("not a regular file")
	}
	data, err := io.ReadAll(io.LimitReader(f, MaxFileSize+1))
	if err != nil {
		return nil, err
	}
	if len(data) > MaxFileSize {
		return nil, ErrTooLarge
	}
	return bytes.TrimPrefix(data, utf8BOM), nil
}

// LineAt returns the 1-based line number of byte offset off.
func LineAt(data []byte, off int) int {
	if off < 0 {
		return 0
	}
	if off > len(data) {
		off = len(data)
	}
	return bytes.Count(data[:off], []byte{'\n'}) + 1
}
