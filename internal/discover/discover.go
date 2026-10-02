// Package discover walks a project tree for dependency manifests.
package discover

import (
	"context"
	"io/fs"
	"path"
	"path/filepath"

	"github.com/omni-line/typosquat-detector/internal/match"
)

var skipDirs = map[string]struct{}{
	"node_modules": {}, "vendor": {}, ".git": {}, "dist": {}, "build": {},
	".venv": {}, "venv": {}, "__pycache__": {}, ".tox": {}, ".mypy_cache": {},
	".pytest_cache": {}, ".svn": {}, ".hg": {}, ".nox": {}, "site-packages": {},
}

// Manifest is a discovered dependency file.
type Manifest struct {
	Path      string
	Rel       string
	Ecosystem string
}

// Options configures Walk.
type Options struct {
	// Classify returns the ecosystem name for a slash-separated path relative
	// to the root, or "" if the file is not a manifest.
	Classify func(rel string) string
	Exclude  *match.Matcher
	// Warn, if set, receives non-fatal problems such as unreadable
	// directories or symlinked manifests that were skipped.
	Warn func(rel string, err error)
}

// Walk finds manifests under root. Symlinks are never followed.
func Walk(ctx context.Context, root string, opts Options) ([]Manifest, error) {
	var out []Manifest
	warn := func(rel string, err error) {
		if opts.Warn != nil {
			opts.Warn(rel, err)
		}
	}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if walkErr != nil {
			if p == root {
				return walkErr
			}
			warn(rel, walkErr)
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		excluded := opts.Exclude != nil && (opts.Exclude.Match(rel) || opts.Exclude.Match(d.Name()))
		if d.IsDir() {
			if _, skip := skipDirs[d.Name()]; skip && p != root {
				return filepath.SkipDir
			}
			if excluded && p != root {
				return filepath.SkipDir
			}
			return nil
		}
		if excluded || opts.Classify == nil {
			return nil
		}
		if rel == "." {
			rel = path.Base(filepath.ToSlash(p))
		}
		eco := opts.Classify(rel)
		if eco == "" {
			return nil
		}
		if !d.Type().IsRegular() {
			warn(rel, errNotRegular(d.Type()))
			return nil
		}
		out = append(out, Manifest{Path: p, Rel: rel, Ecosystem: eco})
		return nil
	})
	return out, err
}

type errNotRegular fs.FileMode

func (e errNotRegular) Error() string {
	m := fs.FileMode(e)
	switch {
	case m&fs.ModeSymlink != 0:
		return "symlinked manifest skipped (symlinks are not followed)"
	case m&fs.ModeNamedPipe != 0:
		return "named pipe skipped (not a regular file)"
	case m&fs.ModeSocket != 0:
		return "socket skipped (not a regular file)"
	case m&fs.ModeDevice != 0:
		return "device file skipped (not a regular file)"
	default:
		return "non-regular file skipped"
	}
}
