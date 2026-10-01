package corpus_test

import (
	"testing"

	"github.com/omni-line/typosquat-detector/internal/corpus"
)

func TestEmbedded(t *testing.T) {
	meta := corpus.MetaInfo()
	if meta.PyPICount < 1000 || meta.NPMCount < 100 {
		t.Fatalf("unexpected meta: %+v", meta)
	}
	npm := corpus.NPM()
	if !npm.Contains("react-dom") || !npm.Contains("cross-env") {
		t.Fatal("npm corpus missing classic targets")
	}
	pypi := corpus.PyPI()
	if !pypi.Contains("requests") {
		t.Fatal("pypi corpus missing requests")
	}
	cands := npm.Candidates("crossenv", 2)
	found := false
	for _, c := range cands {
		if c == "cross-env" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("candidates for crossenv missing cross-env (got %d)", len(cands))
	}
}
