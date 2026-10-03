package ecosystem

import (
	"net/url"
	"path"
	"strings"

	"github.com/omni-line/typosquat-detector/internal/manifest"
	"github.com/omni-line/typosquat-detector/internal/manifest/docker"
)

// Docker checks Dockerfiles and Compose files against popular image names.
func Docker() Ecosystem {
	return Ecosystem{
		Name:     "docker",
		PURLType: "docker",
		IsManifest: func(rel string) bool {
			base := path.Base(rel)
			lower := strings.ToLower(base)
			if lower == "dockerfile" || strings.HasPrefix(lower, "dockerfile.") {
				return true
			}
			return lower == "compose.yaml" || lower == "compose.yml" ||
				lower == "docker-compose.yml" || lower == "docker-compose.yaml"
		},
		Parse: func(rel string, data []byte) ([]manifest.Dependency, error) {
			base := strings.ToLower(path.Base(rel))
			if strings.Contains(base, "compose") {
				return docker.ParseCompose(data)
			}
			return docker.ParseDockerfile(data)
		},
		Normalize: docker.Normalize,
		PackageURL: func(name string) string {
			n := docker.Normalize(name)
			if !strings.Contains(n, "/") {
				return "https://hub.docker.com/_/" + n
			}
			return "https://hub.docker.com/r/" + n
		},
		PURL: func(name string) string {
			return "pkg:docker/" + url.PathEscape(docker.Normalize(name))
		},
	}
}
