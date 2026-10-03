package docker_test

import (
	"testing"

	"github.com/omni-line/typosquat-detector/internal/manifest/docker"
)

func TestParseDockerfile(t *testing.T) {
	deps, err := docker.ParseDockerfile([]byte("FROM nginx:alpine\nFROM docker.io/library/redis:7\nFROM scratch\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(deps) != 2 {
		t.Fatalf("%+v", deps)
	}
	if docker.Normalize("docker.io/library/Nginx:latest") != "nginx" {
		t.Fatal(docker.Normalize("docker.io/library/Nginx:latest"))
	}
}

func TestParseCompose(t *testing.T) {
	deps, err := docker.ParseCompose([]byte("services:\n  db:\n    image: postgres:15\n"))
	if err != nil || len(deps) != 1 || deps[0].Name != "postgres" {
		t.Fatalf("%+v %v", deps, err)
	}
}

func FuzzParseDockerfile(f *testing.F) {
	f.Add([]byte("FROM nginx\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = docker.ParseDockerfile(data)
	})
}
