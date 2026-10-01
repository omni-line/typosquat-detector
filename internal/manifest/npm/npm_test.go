package npm_test

import (
	"testing"

	"github.com/omni-line/typosquat-detector/internal/manifest/npm"
)

func TestParse(t *testing.T) {
	data := []byte(`{
  "dependencies": {
    "lodash": "^4.17.21",
    "local": "file:../local",
    "alias": "npm:react@18"
  },
  "devDependencies": {
    "jest": "29.0.0"
  }
}`)
	deps, err := npm.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, d := range deps {
		got[d.Name] = d.Version
	}
	if got["lodash"] != "^4.17.21" {
		t.Fatalf("lodash: %+v", got)
	}
	if got["react"] != "18" {
		t.Fatalf("alias react: %+v", got)
	}
	if _, ok := got["local"]; ok {
		t.Fatal("file: deps should be skipped")
	}
	if got["jest"] != "29.0.0" {
		t.Fatalf("jest: %+v", got)
	}
}

func TestScope(t *testing.T) {
	if s := npm.Scope("@acme/auth"); s != "@acme" {
		t.Fatalf("got %q", s)
	}
	if s := npm.Scope("lodash"); s != "" {
		t.Fatalf("got %q", s)
	}
}
