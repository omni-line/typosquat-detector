package corpus_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/omni-line/typosquat-detector/internal/corpus"
)

func TestMeta(t *testing.T) {
	meta, err := corpus.MetaInfo()
	if err != nil {
		t.Fatal(err)
	}
	if meta.GeneratedAt == "" {
		t.Fatal("missing generated_at")
	}
	if meta.Ecosystems["pypi"].Count < 1000 || meta.Ecosystems["npm"].Count < 100 {
		t.Fatalf("unexpected meta: %+v", meta)
	}
}

func TestEmbedded(t *testing.T) {
	npm, err := corpus.Load("npm", strings.ToLower)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"react-dom", "cross-env", "jsonstream"} {
		if !npm.Contains(name) {
			t.Errorf("npm corpus missing %q", name)
		}
	}
	if got := npm.Display("jsonstream"); got != "JSONStream" {
		t.Errorf("Display(jsonstream)=%q want JSONStream", got)
	}
	pypi, err := corpus.Load("pypi", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !pypi.Contains("requests") {
		t.Fatal("pypi corpus missing requests")
	}

	found := false
	npm.Candidates("crossenv", 1, func(c string) {
		if c == "cross-env" {
			found = true
		}
	})
	if !found {
		t.Fatal("candidates for crossenv missing cross-env")
	}
}

func TestLoadUnknown(t *testing.T) {
	if _, err := corpus.Load("nope", nil); !errors.Is(err, corpus.ErrUnknown) {
		t.Fatalf("err=%v want ErrUnknown", err)
	}
}

func TestNewSetNormalizes(t *testing.T) {
	s := corpus.NewSet([]string{"PyYAML", "pyyaml", "", "Django"}, strings.ToLower)
	if s.Len() != 2 {
		t.Fatalf("Len=%d want 2", s.Len())
	}
	if !s.Contains("django") || s.Display("django") != "Django" {
		t.Fatalf("normalization lost display name")
	}
}
