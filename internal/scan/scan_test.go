package scan_test

import (
	"strings"
	"testing"

	"github.com/omni-line/typosquat-detector/internal/ecosystem"
	"github.com/omni-line/typosquat-detector/internal/scan"
)

func TestClassicTypos(t *testing.T) {
	res, err := scan.Run("../../testdata", scan.Options{
		Ecosystems:  ecosystem.Default(),
		MaxDistance: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	byPkg := map[string]scan.Finding{}
	for _, f := range res.Findings {
		byPkg[f.Package] = f
	}
	want := map[string]string{
		"crossenv":    "cross-env",
		"react-domm":  "react-dom",
		"reqeusts":    "requests",
		"@acme/authh": "@acme/auth",
	}
	for pkg, sug := range want {
		f, ok := byPkg[pkg]
		if !ok {
			t.Errorf("missing finding for %q; got packages %v", pkg, keys(byPkg))
			continue
		}
		if f.Severity != "critical" {
			t.Errorf("%s severity=%q", pkg, f.Severity)
		}
		if !contains(f.Suggestions, sug) {
			t.Errorf("%s suggestions=%v want %q", pkg, f.Suggestions, sug)
		}
	}
	if _, ok := byPkg["lodash"]; ok {
		t.Error("exact corpus hit lodash should not be a finding")
	}
	if _, ok := byPkg["requests"]; ok {
		t.Error("exact corpus hit requests should not be a finding")
	}
}

func TestAllowSuppresses(t *testing.T) {
	res, err := scan.Run("../../testdata/npm-typos", scan.Options{
		Ecosystems:   ecosystem.Default(),
		MaxDistance:  2,
		Allow:        map[string]struct{}{"crossenv": {}, "react-domm": {}},
		NoScopePeers: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range res.Findings {
		if f.Package == "crossenv" || f.Package == "react-domm" {
			t.Fatalf("allowed package still reported: %+v", f)
		}
	}
}

func TestDistanceOne(t *testing.T) {
	res, err := scan.Run("../../testdata/npm-typos", scan.Options{
		Ecosystems:   ecosystem.Default(),
		MaxDistance:  1,
		NoScopePeers: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	byPkg := map[string]scan.Finding{}
	for _, f := range res.Findings {
		byPkg[f.Package] = f
	}
	if _, ok := byPkg["react-domm"]; !ok {
		t.Error("react-domm distance 1 should flag at --distance 1")
	}
	// crossenv vs cross-env is separator distance 1
	if _, ok := byPkg["crossenv"]; !ok {
		t.Error("crossenv should flag at distance 1 (separator)")
	}
}

func keys(m map[string]scan.Finding) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want || strings.EqualFold(s, want) {
			return true
		}
	}
	return false
}
