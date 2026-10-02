package ecosystem_test

import (
	"testing"

	"github.com/omni-line/typosquat-detector/internal/ecosystem"
)

func TestDefaultValid(t *testing.T) {
	ecos := ecosystem.Default()
	if err := ecosystem.ValidateAll(ecos); err != nil {
		t.Fatal(err)
	}
	for _, e := range ecos {
		set, err := e.LoadCorpus()
		if err != nil {
			t.Fatalf("%s: %v", e.Name, err)
		}
		if set.Len() == 0 {
			t.Fatalf("%s: empty corpus", e.Name)
		}
	}
}

func TestValidateAll(t *testing.T) {
	if err := ecosystem.ValidateAll(nil); err == nil {
		t.Error("want error for no ecosystems")
	}
	if err := ecosystem.ValidateAll([]ecosystem.Ecosystem{{Name: "x"}}); err == nil {
		t.Error("want error for missing fields")
	}
	npm := ecosystem.NPM()
	if err := ecosystem.ValidateAll([]ecosystem.Ecosystem{npm, npm}); err == nil {
		t.Error("want error for duplicates")
	}
}

func TestIdentifiers(t *testing.T) {
	npm, pypi := ecosystem.NPM(), ecosystem.PyPI()
	cases := []struct{ got, want string }{
		{npm.PackageID("@Acme/Auth"), "pkg:npm/%40acme/auth"},
		{npm.PackageID("lodash"), "pkg:npm/lodash"},
		{npm.URL("@acme/auth"), "https://www.npmjs.com/package/@acme/auth"},
		{pypi.PackageID("PyYAML"), "pkg:pypi/pyyaml"},
		{pypi.URL("Django_REST"), "https://pypi.org/project/django-rest/"},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("got %q want %q", tc.got, tc.want)
		}
	}
}

func TestIsManifest(t *testing.T) {
	pypi := ecosystem.PyPI()
	cases := map[string]bool{
		"pyproject.toml":            true,
		"requirements.txt":          true,
		"requirements-dev.txt":      true,
		"requirements/base.txt":     true,
		"docs/requirements.in":      false,
		"notes.txt":                 false,
		"sub/dir/Requirements.TXT":  true,
		"requirements/README.md":    false,
		"apps/api/pyproject.toml":   true,
		"apps/api/pyproject.toml.j": false,
	}
	for rel, want := range cases {
		if got := pypi.IsManifest(rel); got != want {
			t.Errorf("pypi.IsManifest(%q)=%v want %v", rel, got, want)
		}
	}
	if !ecosystem.NPM().IsManifest("a/b/package.json") {
		t.Error("npm should match package.json")
	}
}
