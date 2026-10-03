package composer_test

import (
	"testing"

	"github.com/omni-line/typosquat-detector/internal/manifest/composer"
)

func TestParse(t *testing.T) {
	deps, err := composer.Parse([]byte(`{"require":{"monolog/monolog":"^3","php":">=8"},"require-dev":{"phpunit/phpunit":"^10"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(deps) != 2 {
		t.Fatalf("got %d", len(deps))
	}
	if composer.Normalize("Monolog/Monolog") != "monolog/monolog" {
		t.Fatal("normalize")
	}
}

func FuzzParse(f *testing.F) {
	f.Add([]byte(`{"require":{"a/b":"1"}}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		deps, err := composer.Parse(data)
		if err != nil {
			return
		}
		for _, d := range deps {
			if d.Name == "" || !containsSlash(d.Name) {
				t.Fatalf("bad dep %+v", d)
			}
		}
	})
}

func containsSlash(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '/' {
			return true
		}
	}
	return false
}
