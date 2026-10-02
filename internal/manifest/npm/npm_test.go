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
    "alias": "npm:react@18",
    "gh": "user/repo#main",
    "ws": "workspace:*",
    "tarball": "https://example.com/x.tgz"
  },
  "devDependencies": {
    "jest": "29.0.0",
    "lodash": "4.0.0"
  }
}`)
	deps, err := npm.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	lines := map[string]int{}
	for _, d := range deps {
		got[d.Name] = d.Version
		lines[d.Name] = d.Line
	}
	want := map[string]string{"lodash": "^4.17.21", "react": "18", "jest": "29.0.0"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s=%q want %q", k, got[k], v)
		}
	}
	if lines["lodash"] != 3 || lines["jest"] != 11 {
		t.Errorf("lines=%v", lines)
	}
}

func TestParseInvalid(t *testing.T) {
	if _, err := npm.Parse([]byte(`{bad`)); err == nil {
		t.Fatal("want error")
	}
	if _, err := npm.Parse([]byte(`{"dependencies": {"a": 1}}`)); err == nil {
		t.Fatal("want error for non-string spec")
	}
}

func TestSplit(t *testing.T) {
	cases := map[string][2]string{
		"@acme/auth": {"@acme", "auth"},
		"lodash":     {"", "lodash"},
		"@/x":        {"", "@/x"},
		"@acme/":     {"", "@acme/"},
	}
	for in, want := range cases {
		s, l := npm.Split(in)
		if s != want[0] || l != want[1] {
			t.Errorf("Split(%q)=(%q,%q) want %v", in, s, l, want)
		}
	}
	if npm.Scope("@acme/auth") != "@acme" {
		t.Error("Scope")
	}
}

func FuzzParse(f *testing.F) {
	f.Add([]byte(`{"dependencies":{"a":"1","b":"npm:c@2"}}`))
	f.Add([]byte(`{"devDependencies":null}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		deps, err := npm.Parse(data)
		if err != nil {
			return
		}
		for _, d := range deps {
			if d.Name == "" {
				t.Fatalf("empty name from %q", data)
			}
		}
	})
}
