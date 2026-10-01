package discover

import (
	"io/fs"
	"path"
	"path/filepath"

	"github.com/omni-line/typosquat-detector/internal/match"
)

var skipDirs = map[string]struct{}{
	"node_modules": {}, "vendor": {}, ".git": {}, "dist": {}, "build": {},
	".venv": {}, "venv": {}, "__pycache__": {}, ".tox": {}, ".mypy_cache": {},
	".pytest_cache": {}, ".svn": {}, ".hg": {},
}

// Manifest is a discovered dependency file.
type Manifest struct {
	Path      string
	Rel       string
	Ecosystem string
}

// Options configures Walk.
type Options struct {
	Classify func(rel string) string
	Exclude  *match.Matcher
}

// Walk finds manifests under root.
func Walk(root string, opts Options) ([]Manifest, error) {
	var out []Manifest
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if p == root {
				return walkErr
			}
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if _, skip := skipDirs[d.Name()]; skip {
				return filepath.SkipDir
			}
			if opts.Exclude != nil && (opts.Exclude.Match(rel) || opts.Exclude.Match(d.Name())) {
				return filepath.SkipDir
			}
			return nil
		}
		if opts.Exclude != nil && (opts.Exclude.Match(rel) || opts.Exclude.Match(d.Name())) {
			return nil
		}
		if opts.Classify == nil {
			return nil
		}
		eco := opts.Classify(rel)
		if eco == "" {
			return nil
		}
		if rel == "." {
			rel = path.Base(filepath.ToSlash(p))
		}
		out = append(out, Manifest{Path: p, Rel: rel, Ecosystem: eco})
		return nil
	})
	return out, err
}
