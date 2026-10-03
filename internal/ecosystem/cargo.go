package ecosystem

import (
	"net/url"
	"path"
	"strings"

	"github.com/omni-line/typosquat-detector/internal/manifest"
	"github.com/omni-line/typosquat-detector/internal/manifest/cargo"
)

// Cargo checks Cargo.toml against the crates.io corpus.
func Cargo() Ecosystem {
	return Ecosystem{
		Name:     "cargo",
		PURLType: "cargo",
		IsManifest: func(rel string) bool {
			return strings.EqualFold(path.Base(rel), "Cargo.toml")
		},
		Parse: func(_ string, data []byte) ([]manifest.Dependency, error) {
			return cargo.Parse(data)
		},
		Normalize: cargo.Normalize,
		PackageURL: func(name string) string {
			return "https://crates.io/crates/" + cargo.Normalize(name)
		},
		PURL: func(name string) string {
			return "pkg:cargo/" + url.PathEscape(cargo.Normalize(name))
		},
	}
}
