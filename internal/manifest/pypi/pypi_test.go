package pypi_test

import (
	"testing"

	"github.com/omni-line/typosquat-detector/internal/manifest"
	"github.com/omni-line/typosquat-detector/internal/manifest/pypi"
)

type dep struct {
	version string
	group   string
	line    int
}

func index(deps []manifest.Dependency) map[string]dep {
	out := map[string]dep{}
	for _, d := range deps {
		out[d.Name] = dep{d.Version, d.Group, d.Line}
	}
	return out
}

func TestParseRequirements(t *testing.T) {
	data := []byte(`requests==2.31.0
# comment
reqeusts>=2  # inline comment
flask==2.0 \
    --hash=sha256:abc
-r other.txt
-e ./local
django[argon2]>=4.2 ; python_version >= "3.10"
pkg @ https://example.com/pkg.whl
git+https://github.com/x/y.git
Requests==1.0
numpy
`)
	deps, err := pypi.ParseRequirements(data)
	if err != nil {
		t.Fatal(err)
	}
	got := index(deps)
	want := map[string]dep{
		"requests": {"2.31.0", "requirements", 1},
		"reqeusts": {"2", "requirements", 3},
		"flask":    {"2.0", "requirements", 4},
		"django":   {"4.2", "requirements", 8},
		"numpy":    {"", "requirements", 12},
	}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for name, w := range want {
		if got[name] != w {
			t.Errorf("%s=%+v want %+v", name, got[name], w)
		}
	}
}

func TestParsePyProject(t *testing.T) {
	data := []byte(`[build-system]
requires = ["setuptools>=61", "wheel"]

[project]
name = "x"
dependencies = [
  "django>=4", # don't remove: it's needed
  "reqeusts",
]

[project.optional-dependencies]
test = ["pytest"]

[dependency-groups]
dev = [{include-group = "lint"}, "ruff"]

[tool.poetry.dependencies]
python = "^3.10"
httpx = "^0.27"
local = { path = "../local" }
pydantic = { version = "^2.0", extras = ["email"] }

[tool.poetry.group.docs.dependencies]
mkdocs = "*"

[tool.other]
dependencies = ["ignored"]
`)
	deps, err := pypi.ParsePyProject(data)
	if err != nil {
		t.Fatal(err)
	}
	got := index(deps)
	want := map[string]dep{
		"setuptools": {"61", "build-system", 2},
		"wheel":      {"", "build-system", 2},
		"django":     {"4", "dependencies", 7},
		"reqeusts":   {"", "dependencies", 8},
		"pytest":     {"", "optional-dependencies", 12},
		"ruff":       {"", "dependency-groups", 15},
		"httpx":      {"0.27", "poetry", 19},
		"pydantic":   {"2.0", "poetry", 21},
		"mkdocs":     {"*", "poetry", 24},
	}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for name, w := range want {
		if got[name] != w {
			t.Errorf("%s=%+v want %+v", name, got[name], w)
		}
	}
}

func TestNormalize(t *testing.T) {
	cases := map[string]string{"PyYAML": "pyyaml", "Django_REST": "django-rest", "a.-_b": "a-b"}
	for in, want := range cases {
		if g := pypi.Normalize(in); g != want {
			t.Errorf("Normalize(%q)=%q want %q", in, g, want)
		}
	}
}

func FuzzParseRequirements(f *testing.F) {
	f.Add([]byte("requests==2\nflask \\\n --hash=x\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		deps, _ := pypi.ParseRequirements(data)
		for _, d := range deps {
			if d.Name == "" || d.Line < 1 {
				t.Fatalf("bad dep %+v from %q", d, data)
			}
		}
	})
}

func FuzzParsePyProject(f *testing.F) {
	f.Add([]byte("[project]\ndependencies = [\"a\", # it's\n 'b']\n"))
	f.Add([]byte("[tool.poetry.dependencies]\na = {version = \"1\"}\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		deps, err := pypi.ParsePyProject(data)
		if err != nil {
			return
		}
		for _, d := range deps {
			if d.Name == "" || d.Line < 1 {
				t.Fatalf("bad dep %+v from %q", d, data)
			}
		}
	})
}
