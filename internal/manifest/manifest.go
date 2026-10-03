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

// MaxFileSize is the largest manifest the scanner will read.
const MaxFileSize = 10 << 20

var (
	// ErrTooLarge is returned for manifests over MaxFileSize.
	ErrTooLarge = fmt.Errorf("manifest exceeds %d MiB limit", MaxFileSize>>20)
	// ErrNotRegular is returned for symlinks, FIFOs, devices and other
	// non-regular files.
	ErrNotRegular = errors.New("not a regular file")
)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// ReadFile reads a regular file capped at MaxFileSize and strips a UTF-8 BOM.
//
// Scanned trees are untrusted input: symlinks are refused so reads cannot
// escape the tree, and FIFOs/devices are refused so a crafted checkout cannot
// block the scan forever.
func ReadFile(path string) ([]byte, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !fi.Mode().IsRegular() {
		return nil, ErrNotRegular
	}
	if fi.Size() > MaxFileSize {
		return nil, ErrTooLarge
	}
	f, err := openNoFollow(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !opened.Mode().IsRegular() || !os.SameFile(fi, opened) {
		return nil, ErrNotRegular
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

// JSONKeyLine returns the 1-based line of `"key"` after the first `"section"`
// occurrence in a JSON object. Returns 0 when either needle is missing.
func JSONKeyLine(data []byte, section, key string) int {
	sec := []byte(`"` + section + `"`)
	start := bytes.Index(data, sec)
	if start < 0 {
		return 0
	}
	needle := []byte(`"` + key + `"`)
	off := bytes.Index(data[start:], needle)
	if off < 0 {
		return 0
	}
	return LineAt(data, start+off)
}
