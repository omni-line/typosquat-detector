package rubygems_test

import (
	"testing"

	"github.com/omni-line/typosquat-detector/internal/manifest/rubygems"
)

func TestParse(t *testing.T) {
	deps, err := rubygems.Parse([]byte("gem 'rails', '~> 7'\ngem \"sidekiq\"\ngem 'x', path: '../x'\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(deps) != 2 {
		t.Fatalf("%+v", deps)
	}
}

func FuzzParse(f *testing.F) {
	f.Add([]byte("gem 'a'\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = rubygems.Parse(data)
	})
}
