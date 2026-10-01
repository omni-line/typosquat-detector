package distance_test

import (
	"testing"

	"github.com/omni-line/typosquat-detector/internal/distance"
)

func TestLevenshtein(t *testing.T) {
	cases := []struct {
		a, b string
		d    int
	}{
		{"requests", "requests", 0},
		{"requests", "reqeusts", 2},
		{"react-dom", "react-domm", 1},
		{"", "abc", 3},
	}
	for _, tc := range cases {
		if got := distance.Levenshtein(tc.a, tc.b); got != tc.d {
			t.Errorf("Levenshtein(%q,%q)=%d want %d", tc.a, tc.b, got, tc.d)
		}
	}
}

func TestSeparatorInsensitive(t *testing.T) {
	if d := distance.Distance("crossenv", "cross-env"); d != 1 {
		t.Fatalf("crossenv vs cross-env: %d want 1", d)
	}
	if d := distance.Distance("cross_env", "cross-env"); d != 1 {
		t.Fatalf("cross_env vs cross-env: %d want 1", d)
	}
}
