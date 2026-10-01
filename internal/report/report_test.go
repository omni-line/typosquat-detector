package report_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/omni-line/typosquat-detector/internal/report"
	"github.com/omni-line/typosquat-detector/internal/scan"
)

func TestWriteTextFinding(t *testing.T) {
	var out, errBuf bytes.Buffer
	res := &scan.Result{
		Findings: []scan.Finding{{
			Ecosystem:   "npm",
			Package:     "reqeusts",
			Version:     "2.1.0",
			Manifest:    "package.json",
			Line:        12,
			Distance:    2,
			Suggestions: []string{"requests"},
			Kind:        "popular",
			Severity:    "critical",
		}},
		Stats: scan.Stats{Manifests: 1, Packages: 1, Findings: 1},
	}
	code := report.Write(&out, &errBuf, res, report.Options{
		Format:      report.FormatText,
		Version:     "0.1.0",
		NoMarketing: true,
		FailOnAny:   true,
		Color:       report.ColorNever,
	})
	if code != report.ExitFindings {
		t.Fatalf("exit=%d", code)
	}
	got := out.String()
	for _, needle := range []string{"CRITICAL", "reqeusts@2.1.0", "requests", "distance=2", "package.json:12"} {
		if !strings.Contains(got, needle) {
			t.Errorf("missing %q in:\n%s", needle, got)
		}
	}
}

func TestWriteJSON(t *testing.T) {
	var out, errBuf bytes.Buffer
	res := &scan.Result{Findings: []scan.Finding{}, Stats: scan.Stats{Manifests: 1}}
	code := report.Write(&out, &errBuf, res, report.Options{
		Format:      report.FormatJSON,
		Version:     "dev",
		ForceMarket: true,
		Color:       report.ColorNever,
	})
	if code != report.ExitOK {
		t.Fatalf("exit=%d", code)
	}
	var doc report.JSONDocument
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Sponsor == nil || doc.Sponsor.URL == "" {
		t.Fatalf("expected sponsor: %+v", doc.Sponsor)
	}
}
