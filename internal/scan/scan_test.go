package scan_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/omni-line/typosquat-detector/internal/ecosystem"
	"github.com/omni-line/typosquat-detector/internal/match"
	"github.com/omni-line/typosquat-detector/internal/scan"
)

func run(t *testing.T, root string, opts scan.Options) *scan.Result {
	t.Helper()
	if opts.Ecosystems == nil {
		opts.Ecosystems = ecosystem.Default()
	}
	res, err := scan.Run(context.Background(), root, opts)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func tree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func byPackage(res *scan.Result) map[string]scan.Finding {
	out := map[string]scan.Finding{}
	for _, f := range res.Findings {
		out[f.Package] = f
	}
	return out
}

func TestClassicTypos(t *testing.T) {
	res := run(t, "../../testdata", scan.Options{MaxDistance: 2})
	got := byPackage(res)
	want := map[string]struct {
		sug  string
		sev  scan.Severity
		kind scan.Kind
	}{
		"crossenv":    {"cross-env", scan.SeverityCritical, scan.KindPopular},
		"react-domm":  {"react-dom", scan.SeverityCritical, scan.KindPopular},
		"reqeusts":    {"requests", scan.SeverityCritical, scan.KindPopular},
		"@acme/authh": {"@acme/auth", scan.SeverityHigh, scan.KindScopePeer},
	}
	for pkg, w := range want {
		f, ok := got[pkg]
		if !ok {
			t.Errorf("missing finding for %q; got %v", pkg, res.Findings)
			continue
		}
		if f.Severity != w.sev || f.Kind != w.kind {
			t.Errorf("%s: severity=%s kind=%s want %s/%s", pkg, f.Severity, f.Kind, w.sev, w.kind)
		}
		if len(f.Suggestions) == 0 || f.Suggestions[0] != w.sug {
			t.Errorf("%s: suggestions=%v want %q", pkg, f.Suggestions, w.sug)
		}
		if f.Message == "" || f.Technique == "" || f.Line == 0 {
			t.Errorf("%s: missing detail: %+v", pkg, f)
		}
	}
	for _, exact := range []string{"lodash", "requests", "@acme/auth", "urllib3"} {
		if _, ok := got[exact]; ok {
			t.Errorf("exact package %q should not be a finding", exact)
		}
	}
	if res.Stats.BySeverity[scan.SeverityCritical] < 3 {
		t.Errorf("by_severity=%v", res.Stats.BySeverity)
	}
	if got["crossenv"].PURL != "pkg:npm/crossenv" || got["crossenv"].SuggestionURL != "https://www.npmjs.com/package/cross-env" {
		t.Errorf("identifiers: %+v", got["crossenv"])
	}
}

func TestEveryLocationReported(t *testing.T) {
	root := tree(t, map[string]string{
		"a/package.json": `{"dependencies":{"crossenv":"1"}}`,
		"b/package.json": `{"dependencies":{"crossenv":"1"}}`,
	})
	res := run(t, root, scan.Options{})
	if len(res.Findings) != 2 {
		t.Fatalf("want one finding per manifest, got %+v", res.Findings)
	}
	if res.Findings[0].Manifest != "a/package.json" || res.Findings[1].Manifest != "b/package.json" {
		t.Fatalf("unexpected order: %+v", res.Findings)
	}
}

func TestShortNamesNotFlaggedAtDistanceTwo(t *testing.T) {
	root := tree(t, map[string]string{
		"package.json": `{"dependencies":{"@acme/ui":"1","@acme/api":"1","abc":"1"}}`,
	})
	res := run(t, root, scan.Options{})
	for _, f := range res.Findings {
		if f.Distance > 1 {
			t.Errorf("short name flagged at distance %d: %+v", f.Distance, f)
		}
	}
	if _, ok := byPackage(res)["@acme/api"]; ok {
		t.Error("@acme/api vs @acme/ui is noise and should not be flagged")
	}
}

func TestKnownLegitimatePatterns(t *testing.T) {
	root := tree(t, map[string]string{
		"package.json": `{"dependencies":{
			"@types/bcryptjs":"1",
			"@types/reacct":"1",
			"@fastify/swagger":"1",
			"@fastify/swagger-ui":"1",
			"@acme/ui":"1",
			"@acme/ui-kit":"1"
		}}`,
	})
	got := byPackage(run(t, root, scan.Options{}))
	for _, pkg := range []string{"@types/bcryptjs", "@fastify/swagger-ui", "@fastify/swagger", "@acme/ui-kit"} {
		if f, ok := got[pkg]; ok {
			t.Errorf("legitimate package flagged: %+v", f)
		}
	}
	if _, ok := got["@types/reacct"]; !ok {
		t.Error("@types typo of a popular package should still be flagged")
	}
}

func TestAllowAndIgnoreSuppressAllKinds(t *testing.T) {
	res := run(t, "../../testdata/npm-typos", scan.Options{
		Allow:      map[string]struct{}{"crossenv": {}, "@acme/authh": {}},
		SafeIgnore: match.New("react-*"),
	})
	for _, f := range res.Findings {
		t.Errorf("suppressed package still reported: %+v", f)
	}
	if res.Stats.Skipped != 3 {
		t.Errorf("skipped=%d want 3", res.Stats.Skipped)
	}
}

func TestDistanceOne(t *testing.T) {
	res := run(t, "../../testdata/npm-typos", scan.Options{MaxDistance: 1, NoScopePeers: true})
	got := byPackage(res)
	for _, pkg := range []string{"react-domm", "crossenv"} {
		if _, ok := got[pkg]; !ok {
			t.Errorf("%s should flag at --distance 1", pkg)
		}
	}
	if _, ok := got["@acme/authh"]; ok {
		t.Error("scope peers disabled")
	}
}

func TestExplicitScopeWithPeersDisabled(t *testing.T) {
	res := run(t, "../../testdata/npm-typos", scan.Options{NoScopePeers: true, Scopes: []string{"@acme"}})
	if _, ok := byPackage(res)["@acme/authh"]; !ok {
		t.Error("explicit --scope should still be checked")
	}
}

func TestBrokenManifestIsWarningNotFatal(t *testing.T) {
	root := tree(t, map[string]string{
		"ok/package.json":  `{"dependencies":{"crossenv":"1"}}`,
		"bad/package.json": `{not json`,
	})
	res := run(t, root, scan.Options{})
	if len(res.Findings) != 1 {
		t.Fatalf("findings=%+v", res.Findings)
	}
	if len(res.Warnings) != 1 || res.Warnings[0].Manifest != "bad/package.json" || res.Stats.Warnings != 1 {
		t.Fatalf("warnings=%+v", res.Warnings)
	}
}

func TestCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := scan.Run(ctx, "../../testdata", scan.Options{Ecosystems: ecosystem.Default()})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
}

func TestParseSeverity(t *testing.T) {
	if s, err := scan.ParseSeverity(" HIGH "); err != nil || s != scan.SeverityHigh {
		t.Fatalf("got %q %v", s, err)
	}
	if _, err := scan.ParseSeverity("low"); err == nil {
		t.Fatal("want error")
	}
	if !scan.SeverityCritical.AtLeast(scan.SeverityHigh) || scan.SeverityMedium.AtLeast(scan.SeverityHigh) {
		t.Fatal("AtLeast ordering")
	}
}
