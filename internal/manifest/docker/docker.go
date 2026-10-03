// Package docker parses Dockerfile FROM lines and Compose image fields.
package docker

import (
	"bufio"
	"regexp"
	"strings"

	"github.com/omni-line/typosquat-detector/internal/manifest"
)

var (
	fromRe  = regexp.MustCompile(`(?i)^\s*FROM\s+(?:--[^\s]+\s+)*([^\s]+)`)
	imageRe = regexp.MustCompile(`(?i)^\s*image:\s*["']?([^\s"']+)["']?`)
)

// ParseDockerfile extracts image repository names from Dockerfile FROM lines.
func ParseDockerfile(data []byte) ([]manifest.Dependency, error) {
	return parseLines(data, "FROM", fromRe)
}

// ParseCompose extracts image repository names from Compose YAML image: fields.
func ParseCompose(data []byte) ([]manifest.Dependency, error) {
	return parseLines(data, "image", imageRe)
}

func parseLines(data []byte, group string, re *regexp.Regexp) ([]manifest.Dependency, error) {
	seen := map[string]struct{}{}
	var out []manifest.Dependency
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	lineNo := 0
	for sc.Scan() {
		lineNo++
		m := re.FindStringSubmatch(sc.Text())
		if m == nil {
			continue
		}
		name := Normalize(m[1])
		if name == "" || name == "scratch" {
			continue
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, manifest.Dependency{Name: name, Group: group, Line: lineNo})
	}
	return out, sc.Err()
}

// Normalize strips registry host, tag, digest, and library/ prefix.
func Normalize(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	// Drop digest.
	if i := strings.Index(ref, "@"); i >= 0 {
		ref = ref[:i]
	}
	// Drop tag (last : after last /).
	if i := strings.LastIndex(ref, ":"); i >= 0 {
		if j := strings.LastIndex(ref, "/"); j < i {
			ref = ref[:i]
		}
	}
	// Drop well-known registry hosts.
	for _, host := range []string{"docker.io/", "index.docker.io/", "registry-1.docker.io/"} {
		if strings.HasPrefix(strings.ToLower(ref), host) {
			ref = ref[len(host):]
			break
		}
	}
	ref = strings.TrimPrefix(ref, "library/")
	return strings.ToLower(strings.TrimSpace(ref))
}
