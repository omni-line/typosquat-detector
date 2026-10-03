// Package maven parses pom.xml dependency coordinates.
package maven

import (
	"bytes"
	"encoding/xml"
	"io"
	"strings"

	"github.com/omni-line/typosquat-detector/internal/manifest"
)

type pom struct {
	Dependencies         deps `xml:"dependencies"`
	DependencyManagement struct {
		Dependencies deps `xml:"dependencies"`
	} `xml:"dependencyManagement"`
}

type deps struct {
	Dependency []dep `xml:"dependency"`
}

type dep struct {
	GroupID    string `xml:"groupId"`
	ArtifactID string `xml:"artifactId"`
	Version    string `xml:"version"`
	Scope      string `xml:"scope"`
}

// Parse extracts groupId:artifactId coordinates from a Maven POM.
func Parse(data []byte) ([]manifest.Dependency, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false
	var p pom
	if err := dec.Decode(&p); err != nil && err != io.EOF {
		return nil, err
	}
	seen := map[string]struct{}{}
	var out []manifest.Dependency
	add := func(d dep, group string) {
		g, a := strings.TrimSpace(d.GroupID), strings.TrimSpace(d.ArtifactID)
		if g == "" || a == "" || strings.Contains(g, "${") || strings.Contains(a, "${") {
			return
		}
		name := g + ":" + a
		key := Normalize(name)
		if _, dup := seen[key]; dup {
			return
		}
		seen[key] = struct{}{}
		out = append(out, manifest.Dependency{
			Name:    name,
			Version: strings.TrimSpace(d.Version),
			Group:   group,
			Line:    lineOf(data, a),
		})
	}
	for _, d := range p.Dependencies.Dependency {
		add(d, "dependencies")
	}
	for _, d := range p.DependencyManagement.Dependencies.Dependency {
		add(d, "dependencyManagement")
	}
	return out, nil
}

func lineOf(data []byte, artifactID string) int {
	needle := []byte("<artifactId>" + artifactID + "</artifactId>")
	off := bytes.Index(data, needle)
	if off < 0 {
		return 0
	}
	return manifest.LineAt(data, off)
}

// Normalize lowercases groupId:artifactId.
func Normalize(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// Split returns groupId and artifactId.
func Split(name string) (namespace, leaf string) {
	name = Normalize(name)
	i := strings.IndexByte(name, ':')
	if i <= 0 || i == len(name)-1 {
		return "", name
	}
	return name[:i], name[i+1:]
}
