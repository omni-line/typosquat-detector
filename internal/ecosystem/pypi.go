package ecosystem

import (
	"net/url"
	"path"
	"strings"

	"github.com/omni-line/typosquat-detector/internal/manifest"
	"github.com/omni-line/typosquat-detector/internal/manifest/pypi"
)

// PyPI checks requirements files and pyproject.toml against the PyPI corpus.
func PyPI() Ecosystem {
	return Ecosystem{
		Name:       "pypi",
		PURLType:   "pypi",
		IsManifest: isPythonManifest,
		Parse: func(rel string, data []byte) ([]manifest.Dependency, error) {
			if strings.EqualFold(path.Base(rel), "pyproject.toml") {
				return pypi.ParsePyProject(data)
			}
			return pypi.ParseRequirements(data)
		},
		Normalize: pypi.Normalize,
		PackageURL: func(name string) string {
			return "https://pypi.org/project/" + url.PathEscape(pypi.Normalize(name)) + "/"
		},
		PURL: func(name string) string {
			return "pkg:pypi/" + url.PathEscape(pypi.Normalize(name))
		},
	}
}

func isPythonManifest(rel string) bool {
	base := strings.ToLower(path.Base(rel))
	if base == "pyproject.toml" {
		return true
	}
	if !strings.HasSuffix(base, ".txt") {
		return false
	}
	if strings.HasPrefix(base, "requirements") {
		return true
	}
	return strings.EqualFold(path.Base(path.Dir(rel)), "requirements")
}
