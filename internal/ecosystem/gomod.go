package ecosystem

import (
	"net/url"
	"path"
	"strings"

	"github.com/omni-line/typosquat-detector/internal/manifest"
	"github.com/omni-line/typosquat-detector/internal/manifest/gomod"
)

// GoMod checks go.mod against the popular Go module corpus.
func GoMod() Ecosystem {
	return Ecosystem{
		Name:     "gomod",
		PURLType: "golang",
		IsManifest: func(rel string) bool {
			return strings.EqualFold(path.Base(rel), "go.mod")
		},
		Parse: func(_ string, data []byte) ([]manifest.Dependency, error) {
			return gomod.Parse(data)
		},
		Normalize: gomod.Normalize,
		PackageURL: func(name string) string {
			return "https://pkg.go.dev/" + name
		},
		PURL: func(name string) string {
			return "pkg:golang/" + strings.ReplaceAll(url.PathEscape(gomod.Normalize(name)), "%2F", "/")
		},
	}
}
