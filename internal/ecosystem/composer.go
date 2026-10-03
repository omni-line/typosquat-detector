package ecosystem

import (
	"net/url"
	"path"
	"strings"

	"github.com/omni-line/typosquat-detector/internal/manifest"
	"github.com/omni-line/typosquat-detector/internal/manifest/composer"
)

// Composer checks composer.json against the Packagist corpus.
func Composer() Ecosystem {
	return Ecosystem{
		Name:     "composer",
		PURLType: "composer",
		IsManifest: func(rel string) bool {
			return strings.EqualFold(path.Base(rel), "composer.json")
		},
		Parse: func(_ string, data []byte) ([]manifest.Dependency, error) {
			return composer.Parse(data)
		},
		Normalize: composer.Normalize,
		Namespace: composer.Split,
		PackageURL: func(name string) string {
			return "https://packagist.org/packages/" + composer.Normalize(name)
		},
		PURL: func(name string) string {
			ns, leaf := composer.Split(name)
			if ns == "" {
				return "pkg:composer/" + url.PathEscape(leaf)
			}
			return "pkg:composer/" + url.PathEscape(ns) + "/" + url.PathEscape(leaf)
		},
	}
}
