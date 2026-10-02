package report_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/omni-line/typosquat-detector/internal/report"
	"github.com/omni-line/typosquat-detector/internal/scan"
)

func sample() *scan.Result {
	return &scan.Result{
		Findings: []scan.Finding{{
			Ecosystem:     "pypi",
			Package:       "reqeusts",
			Version:       "2.1.0",
			Manifest:      "requirements.txt",
			Line:          12,
			Group:         "requirements",
			Distance:      1,
			Suggestions:   []string{"requests"},
			Kind:          scan.KindPopular,
			Severity:      scan.SeverityCritical,
			Technique:     "transposition",
			Message:       `"reqeusts" is 1 edit away from a popular pypi package, "requests"`,
			PURL:          "pkg:pypi/reqeusts",
			RegistryURL:   "https://pypi.org/project/reqeusts/",
			SuggestionURL: "https://pypi.org/project/requests/",
		}},
		Warnings: []scan.Warning{{Manifest: "bad/package.json", Message: "parse error"}},
		Stats: scan.Stats{Manifests: 2, Packages: 1, Findings: 1, Warnings: 1,
			BySeverity: map[scan.Severity]int{scan.SeverityCritical: 1}},
	}
}

func TestWriteText(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := report.Write(&out, &errBuf, sample(), report.Options{
		Format:      report.FormatText,
		Version:     "0.1.0",
		NoMarketing: true,
		FailOn:      scan.SeverityMedium,
		Color:       report.ColorNever,
	})
	if code != report.ExitFindings {
		t.Fatalf("exit=%d", code)
	}
	got := out.String()
	for _, needle := range []string{
		"1 typosquat finding:", "1 critical", "CRITICAL", "reqeusts@2.1.0",
		"requirements.txt:12 (requirements)", "did you mean", "requests",
		"two adjacent characters swapped", "https://pypi.org/project/requests/",
		"--allow reqeusts", "bad/package.json: parse error", "1 warning",
	} {
		if !strings.Contains(got, needle) {
			t.Errorf("missing %q in:\n%s", needle, got)
		}
	}
}

func TestWriteQuiet(t *testing.T) {
	var out, errBuf bytes.Buffer
	report.Write(&out, &errBuf, sample(), report.Options{Format: report.FormatText, Quiet: true, Color: report.ColorNever})
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 1 || !strings.HasPrefix(lines[0], "CRITICAL pypi reqeusts@2.1.0 requirements.txt:12") {
		t.Fatalf("quiet output:\n%s", out.String())
	}
	if !strings.Contains(errBuf.String(), "bad/package.json") {
		t.Fatalf("warnings should go to stderr in quiet mode: %q", errBuf.String())
	}
}

func TestSanitizesTerminalEscapes(t *testing.T) {
	res := sample()
	res.Findings[0].Package = "evil\x1b[31m\u202egnp\u200b"
	var out, errBuf bytes.Buffer
	report.Write(&out, &errBuf, res, report.Options{Format: report.FormatText, NoMarketing: true, Color: report.ColorNever})
	got := out.String()
	if strings.ContainsAny(got, "\x1b\u202e\u200b") {
		t.Fatalf("raw control/format characters in output: %q", got)
	}
	if !strings.Contains(got, `evil\u001b[31m\u202egnp\u200b`) {
		t.Fatalf("expected visible escapes in: %s", got)
	}
}

func TestWriteJSON(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := report.Write(&out, &errBuf, sample(), report.Options{
		Format:         report.FormatJSON,
		Version:        "dev",
		ForceMarketing: true,
		Color:          report.ColorNever,
		MaxDistance:    2,
	})
	if code != report.ExitOK {
		t.Fatalf("exit=%d", code)
	}
	var doc report.JSONDocument
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.SchemaVersion != report.JSONSchemaVersion || doc.Scan == nil || doc.Scan.MaxDistance != 2 {
		t.Fatalf("doc=%+v", doc)
	}
	if len(doc.Findings) != 1 || doc.Findings[0].PURL == "" || len(doc.Warnings) != 1 {
		t.Fatalf("findings/warnings: %+v", doc)
	}
	if doc.Sponsor == nil || doc.Sponsor.URL == "" {
		t.Fatalf("expected sponsor: %+v", doc.Sponsor)
	}
}

func TestWriteSARIF(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := report.Write(&out, &errBuf, sample(), report.Options{Format: report.FormatSARIF, Version: "1.2.3"})
	if code != report.ExitOK {
		t.Fatalf("exit=%d", code)
	}
	var doc struct {
		Version string `json:"version"`
		Runs    []struct {
			Tool struct {
				Driver struct {
					Name  string `json:"name"`
					Rules []struct {
						ID         string         `json:"id"`
						Properties map[string]any `json:"properties"`
					} `json:"rules"`
				} `json:"driver"`
			} `json:"tool"`
			Invocations []struct {
				Notifications []any `json:"toolExecutionNotifications"`
			} `json:"invocations"`
			Results []struct {
				RuleID    string `json:"ruleId"`
				RuleIndex int    `json:"ruleIndex"`
				Level     string `json:"level"`
				Locations []struct {
					PhysicalLocation struct {
						ArtifactLocation struct {
							URI string `json:"uri"`
						} `json:"artifactLocation"`
						Region struct {
							StartLine int `json:"startLine"`
						} `json:"region"`
					} `json:"physicalLocation"`
				} `json:"locations"`
				PartialFingerprints map[string]string `json:"partialFingerprints"`
			} `json:"results"`
		} `json:"runs"`
	}
	if err := json.Unmarshal(out.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Version != "2.1.0" || len(doc.Runs) != 1 {
		t.Fatalf("bad envelope: %s", out.String())
	}
	run := doc.Runs[0]
	if len(run.Results) != 1 || len(run.Invocations) != 1 || len(run.Invocations[0].Notifications) != 1 {
		t.Fatalf("results/notifications: %s", out.String())
	}
	r := run.Results[0]
	rule := run.Tool.Driver.Rules[r.RuleIndex]
	if rule.ID != r.RuleID || r.Level != "error" || rule.Properties["security-severity"] == nil {
		t.Fatalf("rule mismatch: %+v vs %+v", r, rule)
	}
	loc := r.Locations[0].PhysicalLocation
	if loc.ArtifactLocation.URI != "requirements.txt" || loc.Region.StartLine != 12 {
		t.Fatalf("location: %+v", loc)
	}
	if len(r.PartialFingerprints) != 1 {
		t.Fatal("missing fingerprint")
	}
	if strings.Contains(out.String(), "omniline.app") {
		t.Fatal("SARIF output must not contain marketing")
	}
}

func TestExitCode(t *testing.T) {
	res := sample()
	cases := []struct {
		opts report.Options
		want int
	}{
		{report.Options{FailOn: scan.SeverityMedium}, report.ExitFindings},
		{report.Options{FailOn: scan.SeverityCritical}, report.ExitFindings},
		{report.Options{}, report.ExitOK},
		{report.Options{Strict: true}, report.ExitError},
	}
	for i, tc := range cases {
		if got := report.ExitCode(res, tc.opts); got != tc.want {
			t.Errorf("case %d: exit=%d want %d", i, got, tc.want)
		}
	}
	res.Findings[0].Severity = scan.SeverityMedium
	if got := report.ExitCode(res, report.Options{FailOn: scan.SeverityHigh}); got != report.ExitOK {
		t.Errorf("medium finding should not fail --fail-on high, got %d", got)
	}
}
