package ecosystem

import (
	"net/url"
	"path"
	"strings"

	"github.com/omni-line/typosquat-detector/internal/manifest"
	"github.com/omni-line/typosquat-detector/internal/manifest/npm"
)

// NPM checks package.json against the npm corpus.
func NPM() Ecosystem {
	return Ecosystem{
		Name:     "npm",
		PURLType: "npm",
		IsManifest: func(rel string) bool {
			return strings.EqualFold(path.Base(rel), "package.json")
		},
		Parse: func(_ string, data []byte) ([]manifest.Dependency, error) {
			return npm.Parse(data)
		},
		Normalize: npm.Normalize,
		Namespace: npm.Split,
		Implied:   impliedNPM,
		PackageURL: func(name string) string {
			return "https://www.npmjs.com/package/" + name
		},
		PURL: func(name string) string {
			scope, leaf := npm.Split(npm.Normalize(name))
			if scope == "" {
				return "pkg:npm/" + url.PathEscape(leaf)
			}
			// purl requires the scope's "@" to be percent-encoded.
			return "pkg:npm/%40" + url.PathEscape(strings.TrimPrefix(scope, "@")) + "/" + url.PathEscape(leaf)
		},
	}
}

// impliedNPM trusts DefinitelyTyped packages for popular libraries.
// DefinitelyTyped encodes "@scope/pkg" as "@types/scope__pkg".
func impliedNPM(key string, popular func(string) bool) bool {
	scope, leaf := npm.Split(key)
	if scope != "@types" {
		return false
	}
	if s, p, ok := strings.Cut(leaf, "__"); ok {
		return popular("@" + s + "/" + p)
	}
	return popular(leaf)
}
