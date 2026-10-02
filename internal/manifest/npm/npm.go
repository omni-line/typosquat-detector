// Package npm parses package.json manifests.
package npm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/omni-line/typosquat-detector/internal/manifest"
)

// Groups are the package.json sections that declare registry dependencies.
var Groups = []string{"dependencies", "devDependencies", "optionalDependencies", "peerDependencies"}

var skipPrefixes = []string{
	"file:", "link:", "workspace:", "portal:", "patch:",
	"git:", "git+", "github:", "gitlab:", "bitbucket:", "gist:",
	"http:", "https:", "./", "../", "/", "~/",
}

// Parse extracts registry-resolved dependencies from package.json.
func Parse(data []byte) ([]manifest.Dependency, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse package.json: %w", err)
	}
	seen := map[string]struct{}{}
	var out []manifest.Dependency
	for _, group := range Groups {
		section, ok := raw[group]
		if !ok || string(section) == "null" {
			continue
		}
		var deps map[string]string
		if err := json.Unmarshal(section, &deps); err != nil {
			return nil, fmt.Errorf("parse package.json %s: %w", group, err)
		}
		keys := make([]string, 0, len(deps))
		for k := range deps {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, key := range keys {
			name, ver, ok := resolve(key, deps[key])
			if !ok || name == "" {
				continue
			}
			if _, dup := seen[name]; dup {
				continue
			}
			seen[name] = struct{}{}
			out = append(out, manifest.Dependency{
				Name:    name,
				Version: ver,
				Group:   group,
				Line:    keyLine(data, group, key),
			})
		}
	}
	return out, nil
}

// resolve returns the registry package a dependency entry installs. Aliases
// ("x": "npm:real@1") resolve to the aliased target, which is what actually
// gets downloaded.
func resolve(key, spec string) (name, version string, ok bool) {
	spec = strings.TrimSpace(spec)
	lower := strings.ToLower(spec)
	if strings.HasPrefix(lower, "npm:") {
		target := spec[len("npm:"):]
		if at := strings.LastIndex(target, "@"); at > 0 {
			return target[:at], target[at+1:], true
		}
		return target, "", true
	}
	for _, p := range skipPrefixes {
		if strings.HasPrefix(lower, p) {
			return "", "", false
		}
	}
	// "user/repo#ref" GitHub shorthand, bare paths, and tarballs.
	if strings.Contains(spec, "/") || strings.HasSuffix(lower, ".tgz") || strings.HasSuffix(lower, ".tar.gz") {
		return "", "", false
	}
	return key, spec, true
}

func keyLine(data []byte, section, key string) int {
	start := bytes.Index(data, []byte(`"`+section+`"`))
	if start < 0 {
		return 0
	}
	off := bytes.Index(data[start:], []byte(`"`+key+`"`))
	if off < 0 {
		return 0
	}
	return manifest.LineAt(data, start+off)
}

// Normalize returns the comparison key for an npm name. The registry only
// accepts lowercase names for new packages, so case is not significant.
func Normalize(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// Scope returns "@org" for scoped packages, or "".
func Scope(name string) string {
	scope, _ := Split(name)
	return scope
}

// Split separates "@org/pkg" into ("@org", "pkg"). Unscoped names return
// ("", name).
func Split(name string) (scope, leaf string) {
	if !strings.HasPrefix(name, "@") {
		return "", name
	}
	i := strings.Index(name, "/")
	if i <= 1 || i+1 >= len(name) {
		return "", name
	}
	return name[:i], name[i+1:]
}
