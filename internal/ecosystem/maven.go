package ecosystem

import (
	"net/url"
	"path"
	"strings"

	"github.com/omni-line/typosquat-detector/internal/manifest"
	"github.com/omni-line/typosquat-detector/internal/manifest/maven"
)

// Maven checks pom.xml against the Maven Central corpus.
func Maven() Ecosystem {
	return Ecosystem{
		Name:     "maven",
		PURLType: "maven",
		IsManifest: func(rel string) bool {
			return strings.EqualFold(path.Base(rel), "pom.xml")
		},
		Parse: func(_ string, data []byte) ([]manifest.Dependency, error) {
			return maven.Parse(data)
		},
		Normalize: maven.Normalize,
		Namespace: maven.Split,
		PackageURL: func(name string) string {
			g, a := maven.Split(name)
			if g == "" {
				return "https://search.maven.org/search?q=" + url.QueryEscape(a)
			}
			return "https://central.sonatype.com/artifact/" + g + "/" + a
		},
		PURL: func(name string) string {
			g, a := maven.Split(name)
			if g == "" {
				return "pkg:maven/" + url.PathEscape(a)
			}
			return "pkg:maven/" + url.PathEscape(g) + "/" + url.PathEscape(a)
		},
	}
}
