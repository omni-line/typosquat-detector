package ecosystem

import (
	"net/url"
	"path"
	"strings"

	"github.com/omni-line/typosquat-detector/internal/manifest"
	"github.com/omni-line/typosquat-detector/internal/manifest/rubygems"
)

// RubyGems checks Gemfile against the rubygems.org corpus.
func RubyGems() Ecosystem {
	return Ecosystem{
		Name:     "rubygems",
		PURLType: "gem",
		IsManifest: func(rel string) bool {
			return strings.EqualFold(path.Base(rel), "Gemfile")
		},
		Parse: func(_ string, data []byte) ([]manifest.Dependency, error) {
			return rubygems.Parse(data)
		},
		Normalize: rubygems.Normalize,
		PackageURL: func(name string) string {
			return "https://rubygems.org/gems/" + rubygems.Normalize(name)
		},
		PURL: func(name string) string {
			return "pkg:gem/" + url.PathEscape(rubygems.Normalize(name))
		},
	}
}
