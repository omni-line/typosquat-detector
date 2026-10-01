package pypi_test

import (
	"testing"

	"github.com/omni-line/typosquat-detector/internal/manifest/pypi"
)

func TestParseRequirements(t *testing.T) {
	data := []byte("requests==2.31.0\n# comment\nreqeusts>=2\n")
	deps, err := pypi.ParseRequirements(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(deps) != 2 {
		t.Fatalf("got %d deps", len(deps))
	}
	if deps[0].Name != "requests" || deps[0].Line != 1 {
		t.Fatalf("%+v", deps[0])
	}
	if deps[1].Name != "reqeusts" {
		t.Fatalf("%+v", deps[1])
	}
}

func TestNormalize(t *testing.T) {
	if g := pypi.Normalize("PyYAML"); g != "pyyaml" {
		t.Fatalf("%q", g)
	}
	if g := pypi.Normalize("Django_REST"); g != "django-rest" {
		t.Fatalf("%q", g)
	}
}
