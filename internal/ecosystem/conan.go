package ecosystem

import (
	"net/url"
	"path"
	"strings"

	"github.com/omni-line/typosquat-detector/internal/manifest"
	"github.com/omni-line/typosquat-detector/internal/manifest/conan"
)

// Conan checks conanfile.txt against the ConanCenter corpus.
func Conan() Ecosystem {
	return Ecosystem{
		Name:     "conan",
		PURLType: "conan",
		IsManifest: func(rel string) bool {
			return strings.EqualFold(path.Base(rel), "conanfile.txt")
		},
		Parse: func(_ string, data []byte) ([]manifest.Dependency, error) {
			return conan.Parse(data)
		},
		Normalize: conan.Normalize,
		PackageURL: func(name string) string {
			return "https://conan.io/center/recipes/" + conan.Normalize(name)
		},
		PURL: func(name string) string {
			return "pkg:conan/" + url.PathEscape(conan.Normalize(name))
		},
	}
}
