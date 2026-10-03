package cargo_test

import (
	"testing"

	"github.com/omni-line/typosquat-detector/internal/manifest/cargo"
)

func TestParse(t *testing.T) {
	deps, err := cargo.Parse([]byte("[dependencies]\nserde = \"1\"\nlocal = { path = \"../x\" }\n[dev-dependencies]\nclap = \"4\"\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(deps) != 2 {
		t.Fatalf("got %+v", deps)
	}
}

func FuzzParse(f *testing.F) {
	f.Add([]byte("[dependencies]\na = \"1\"\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = cargo.Parse(data)
	})
}
