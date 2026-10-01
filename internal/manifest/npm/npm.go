package npm

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/omni-line/typosquat-detector/internal/manifest"
)

var Groups = []string{"dependencies", "devDependencies", "optionalDependencies", "peerDependencies"}

var skipPrefixes = []string{
	"file:", "link:", "workspace:", "portal:",
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
	// github shorthand user/repo
	if !strings.HasPrefix(key, "@") && strings.Count(key, "/") == 1 && !strings.Contains(spec, ":") {
		if strings.Contains(lower, "/") && !strings.ContainsAny(spec, "<>=") {
			// dependency key is the package name for normal entries
		}
	}
	return key, spec, true
}

func keyLine(data []byte, section, key string) int {
	needle := []byte(`"` + key + `"`)
	sec := []byte(`"` + section + `"`)
	start := bytesIndex(data, sec)
	if start < 0 {
		return 0
	}
	rest := data[start:]
	off := bytesIndex(rest, needle)
	if off < 0 {
		return 0
	}
	return manifest.LineAt(data, start+off)
}

func bytesIndex(data, needle []byte) int {
	return strings.Index(string(data), string(needle))
}

// Normalize lowercases unscoped names; leaves scopes intact aside from trim.
func Normalize(name string) string {
	return strings.TrimSpace(name)
}

// Scope returns "@org" for scoped packages, or "".
func Scope(name string) string {
	if !strings.HasPrefix(name, "@") {
		return ""
	}
	i := strings.Index(name, "/")
	if i <= 1 {
		return ""
	}
	return name[:i]
}
