// Package ecosystem binds manifest formats, name normalization, registry
// metadata and popular-name corpora for each supported package registry.
//
// To add a registry: create <name>.go in this package returning an Ecosystem,
// add it to Default, embed internal/corpus/<name>.json.gz, and teach
// scripts/update-corpus where to fetch the top-package list. See
// CONTRIBUTING.md for the full checklist.
package ecosystem

import (
	"errors"
	"fmt"

	"github.com/omni-line/typosquat-detector/internal/corpus"
	"github.com/omni-line/typosquat-detector/internal/manifest"
)

// Ecosystem describes one package registry. Name, IsManifest and Parse are
// required; everything else is optional and degrades gracefully.
type Ecosystem struct {
	// Name is the stable identifier used in output and as the corpus file
	// base name (e.g. "npm" -> internal/corpus/npm.json.gz).
	Name string
	// PURLType is the package-url type (https://github.com/package-url/purl-spec).
	PURLType string

	// IsManifest reports whether a slash-separated relative path is a
	// manifest this ecosystem can parse.
	IsManifest func(rel string) bool
	// Parse extracts direct registry dependencies from a manifest.
	Parse func(rel string, data []byte) ([]manifest.Dependency, error)
	// Normalize maps a name to its registry identity (e.g. PEP 503). Names
	// with equal keys are the same package.
	Normalize func(name string) string
	// Namespace splits a name into an org namespace and leaf (npm scopes).
	// When set, packages sharing a namespace are checked against each other.
	Namespace func(name string) (namespace, leaf string)
	// Implied reports whether a normalized name is legitimate because a
	// related popular package exists (npm @types/x for popular x).
	Implied func(key string, popular func(key string) bool) bool
	// PackageURL returns the public registry web page for a package.
	PackageURL func(name string) string
	// PURL returns the package-url for a package without a version.
	PURL func(name string) string
	// Corpus returns the popular-name set. Defaults to the embedded snapshot
	// named after Name, normalized with Normalize.
	Corpus func() (*corpus.Set, error)
}

// Validate checks required fields.
func (e Ecosystem) Validate() error {
	var missing []string
	if e.Name == "" {
		missing = append(missing, "Name")
	}
	if e.IsManifest == nil {
		missing = append(missing, "IsManifest")
	}
	if e.Parse == nil {
		missing = append(missing, "Parse")
	}
	if len(missing) > 0 {
		return fmt.Errorf("ecosystem %q: missing %v", e.Name, missing)
	}
	return nil
}

// Key returns the normalized identity of name.
func (e Ecosystem) Key(name string) string {
	if e.Normalize == nil {
		return name
	}
	return e.Normalize(name)
}

// LoadCorpus returns the popular-name set for e.
func (e Ecosystem) LoadCorpus() (*corpus.Set, error) {
	if e.Corpus != nil {
		return e.Corpus()
	}
	return corpus.Load(e.Name, e.Normalize)
}

// URL returns the registry page for name, or "".
func (e Ecosystem) URL(name string) string {
	if e.PackageURL == nil {
		return ""
	}
	return e.PackageURL(name)
}

// PackageID returns the package-url for name, or "".
func (e Ecosystem) PackageID(name string) string {
	if e.PURL == nil {
		return ""
	}
	return e.PURL(name)
}

// Default returns all built-in ecosystems.
func Default() []Ecosystem {
	return []Ecosystem{
		NPM(), PyPI(), Composer(), GoMod(), Cargo(), Maven(), RubyGems(), Docker(), Conan(),
	}
}

// ValidateAll checks ecosystems and rejects duplicate names.
func ValidateAll(ecos []Ecosystem) error {
	if len(ecos) == 0 {
		return errors.New("no ecosystems configured")
	}
	seen := map[string]struct{}{}
	for _, e := range ecos {
		if err := e.Validate(); err != nil {
			return err
		}
		if _, dup := seen[e.Name]; dup {
			return fmt.Errorf("ecosystem %q registered twice", e.Name)
		}
		seen[e.Name] = struct{}{}
	}
	return nil
}
