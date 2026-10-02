package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/omni-line/typosquat-detector/internal/cli"
	"github.com/omni-line/typosquat-detector/internal/report"
)

func run(args ...string) (code int, stdout, stderr string) {
	var out, errBuf bytes.Buffer
	code = cli.RunContext(context.Background(), args, &out, &errBuf)
	return code, out.String(), errBuf.String()
}

func TestVersion(t *testing.T) {
	code, out, errOut := run("--version")
	if code != report.ExitOK {
		t.Fatalf("code=%d err=%s", code, errOut)
	}
	if strings.TrimSpace(out) == "" {
		t.Fatal("empty version")
	}
}

func TestScanFixtureJSON(t *testing.T) {
	code, out, errOut := run("../../testdata/npm-typos", "--format", "json", "--no-marketing", "--color", "never")
	if code != report.ExitFindings {
		t.Fatalf("code=%d err=%s out=%s", code, errOut, out)
	}
	var doc report.JSONDocument
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Sponsor != nil {
		t.Fatal("--no-marketing should drop sponsor")
	}
	if doc.Scan == nil || doc.Scan.Corpus == nil || doc.Scan.Corpus.GeneratedAt == "" {
		t.Fatalf("missing corpus metadata: %+v", doc.Scan)
	}
	if !strings.Contains(out, "crossenv") {
		t.Fatalf("expected crossenv finding: %s", out)
	}
}

func TestFailOnSeverity(t *testing.T) {
	code, _, _ := run("../../testdata/npm-typos", "--fail-on", "none", "-q")
	if code != report.ExitOK {
		t.Fatalf("--fail-on none code=%d", code)
	}
	code, _, _ = run("../../testdata/npm-typos", "--fail-on", "critical", "-q")
	if code != report.ExitFindings {
		t.Fatalf("--fail-on critical code=%d", code)
	}
}

func TestSARIF(t *testing.T) {
	code, out, _ := run("../../testdata", "--format", "sarif")
	if code != report.ExitFindings || !strings.Contains(out, `"version": "2.1.0"`) {
		t.Fatalf("code=%d out=%s", code, out)
	}
}

func TestInvalidFlags(t *testing.T) {
	for _, args := range [][]string{
		{"--format", "xml"},
		{"--fail-on", "low"},
		{"--distance", "3"},
		{"--color", "rainbow"},
		{"a", "b"},
		{"--format"},
	} {
		if code, _, _ := run(args...); code != report.ExitError {
			t.Errorf("%v: code=%d want %d", args, code, report.ExitError)
		}
	}
}

func TestMissingPath(t *testing.T) {
	code, _, errOut := run("does-not-exist")
	if code != report.ExitError || !strings.Contains(errOut, "does-not-exist") {
		t.Fatalf("code=%d err=%s", code, errOut)
	}
}

func TestInterrupted(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var out, errBuf bytes.Buffer
	code := cli.RunContext(ctx, []string{"../../testdata"}, &out, &errBuf)
	if code != report.ExitError || !strings.Contains(errBuf.String(), "interrupted") {
		t.Fatalf("code=%d err=%s", code, errBuf.String())
	}
}

func TestHelp(t *testing.T) {
	code, _, errOut := run("--help")
	if code != report.ExitOK {
		t.Fatalf("code=%d", code)
	}
	if !strings.Contains(errOut, "Usage:") || !strings.Contains(errOut, "Severities:") {
		t.Fatalf("usage missing: %s", errOut)
	}
}
