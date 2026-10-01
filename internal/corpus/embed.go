// Package corpus loads embedded popular-package name snapshots.
package corpus

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"

	_ "embed"
)

//go:embed npm.json.gz
var npmGZ []byte

//go:embed pypi.json.gz
var pypiGZ []byte

//go:embed meta.json
var metaJSON []byte

// Meta describes when/how the corpus was built.
type Meta struct {
	GeneratedAt string `json:"generated_at"`
	NPMCount    int    `json:"npm_count"`
	PyPICount   int    `json:"pypi_count"`
	NPMSource   string `json:"npm_source"`
	PyPISource  string `json:"pypi_source"`
}

// Set is an indexed popular-name corpus for one ecosystem.
type Set struct {
	exact map[string]struct{}
	byLen map[int][]string
	names []string
}

var (
	loadOnce sync.Once
	npmSet   *Set
	pypiSet  *Set
	meta     Meta
	loadErr  error
)

// MetaInfo returns corpus metadata.
func MetaInfo() Meta {
	ensure()
	return meta
}

// NPM returns the npm popular-name set.
func NPM() *Set {
	ensure()
	return npmSet
}

// PyPI returns the PyPI popular-name set.
func PyPI() *Set {
	ensure()
	return pypiSet
}

func ensure() {
	loadOnce.Do(func() {
		if err := json.Unmarshal(metaJSON, &meta); err != nil {
			loadErr = fmt.Errorf("corpus meta: %w", err)
			return
		}
		npmSet, loadErr = loadGzip(npmGZ)
		if loadErr != nil {
			return
		}
		pypiSet, loadErr = loadGzip(pypiGZ)
	})
	if loadErr != nil {
		panic(loadErr)
	}
}

func loadGzip(raw []byte) (*Set, error) {
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	data, err := io.ReadAll(zr)
	if err != nil {
		return nil, err
	}
	var names []string
	if err := json.Unmarshal(data, &names); err != nil {
		return nil, err
	}
	return NewSet(names), nil
}

// NewSet indexes names for exact and length-bucketed lookup.
func NewSet(names []string) *Set {
	s := &Set{
		exact: make(map[string]struct{}, len(names)),
		byLen: make(map[int][]string),
		names: names,
	}
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		s.exact[n] = struct{}{}
		s.byLen[len(n)] = append(s.byLen[len(n)], n)
	}
	return s
}

// Contains reports whether name is an exact corpus member.
func (s *Set) Contains(name string) bool {
	if s == nil {
		return false
	}
	_, ok := s.exact[name]
	return ok
}

// Candidates returns corpus names whose length is within maxDist of len(name).
func (s *Set) Candidates(name string, maxDist int) []string {
	if s == nil {
		return nil
	}
	l := len(name)
	var out []string
	for d := -maxDist; d <= maxDist; d++ {
		out = append(out, s.byLen[l+d]...)
	}
	return out
}

// Len returns the number of exact names.
func (s *Set) Len() int {
	if s == nil {
		return 0
	}
	return len(s.exact)
}
