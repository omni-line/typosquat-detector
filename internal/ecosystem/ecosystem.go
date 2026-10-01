package ecosystem

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/omni-line/typosquat-detector/internal/corpus"
	"github.com/omni-line/typosquat-detector/internal/manifest"
	"github.com/omni-line/typosquat-detector/internal/manifest/npm"
	"github.com/omni-line/typosquat-detector/internal/manifest/pypi"
)

// Ecosystem binds a manifest format to a popular-name corpus.
type Ecosystem struct {
	Name       string
	IsManifest func(rel string) bool
	Parse      func(rel string, data []byte) ([]manifest.Dependency, error)
	Normalize  func(name string) string
	Corpus     func() *corpus.Set
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
	if e.Corpus == nil {
		missing = append(missing, "Corpus")
	}
	if len(missing) > 0 {
		return fmt.Errorf("ecosystem %q: missing %v", e.Name, missing)
	}
	return nil
}

// Key returns the normalized identity for dedupe.
func (e Ecosystem) Key(name string) string {
	if e.Normalize == nil {
		return name
	}
	return e.Normalize(name)
}

// Default returns npm and PyPI ecosystems.
func Default() []Ecosystem {
	return []Ecosystem{NPM(), PyPI()}
}

// ValidateAll checks ecosystems.
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

// NPM is package.json against the npm corpus.
func NPM() Ecosystem {
	return Ecosystem{
		Name: "npm",
		IsManifest: func(rel string) bool {
			return strings.EqualFold(path.Base(rel), "package.json")
		},
		Parse: func(_ string, data []byte) ([]manifest.Dependency, error) {
			return npm.Parse(data)
		},
		Normalize: npm.Normalize,
		Corpus:    corpus.NPM,
	}
}

// PyPI is requirements / pyproject against the PyPI corpus.
func PyPI() Ecosystem {
	return Ecosystem{
		Name:       "pypi",
		IsManifest: isPythonManifest,
		Parse: func(rel string, data []byte) ([]manifest.Dependency, error) {
			if strings.EqualFold(path.Base(rel), "pyproject.toml") {
				return pypi.ParsePyProject(data)
			}
			return pypi.ParseRequirements(data)
		},
		Normalize: pypi.Normalize,
		Corpus:    corpus.PyPI,
	}
}

func isPythonManifest(rel string) bool {
	base := strings.ToLower(path.Base(rel))
	if base == "pyproject.toml" {
		return true
	}
	if strings.HasSuffix(base, ".txt") && strings.HasPrefix(base, "requirements") {
		return true
	}
	return strings.HasSuffix(base, ".txt") && strings.EqualFold(path.Base(path.Dir(rel)), "requirements")
}
