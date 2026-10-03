package gomod_test

import (
	"testing"

	"github.com/omni-line/typosquat-detector/internal/manifest/gomod"
)

func TestParse(t *testing.T) {
	deps, err := gomod.Parse([]byte("module m\n\nrequire (\n\tgithub.com/gin-gonic/gin v1.9.1\n)\nrequire rsc.io/quote v1.5.2\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(deps) != 2 {
		t.Fatalf("got %+v", deps)
	}
}

func FuzzParse(f *testing.F) {
	f.Add([]byte("require github.com/a/b v1.0.0\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = gomod.Parse(data)
	})
}
