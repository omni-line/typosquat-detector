package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/omni-line/typosquat-detector/internal/cli"
	"github.com/omni-line/typosquat-detector/internal/report"
)

func TestVersion(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := cli.Run([]string{"--version"}, &out, &errBuf)
	if code != report.ExitOK {
		t.Fatalf("code=%d err=%s", code, errBuf.String())
	}
	if strings.TrimSpace(out.String()) == "" {
		t.Fatal("empty version")
	}
}

func TestScanFixture(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := cli.Run([]string{
		"../../testdata/npm-typos",
		"--format", "json",
		"--no-marketing",
		"--color", "never",
	}, &out, &errBuf)
	if code != report.ExitFindings {
		t.Fatalf("code=%d err=%s out=%s", code, errBuf.String(), out.String())
	}
	if !strings.Contains(out.String(), "crossenv") {
		t.Fatalf("expected crossenv finding: %s", out.String())
	}
}

func TestHelp(t *testing.T) {
	var out, errBuf bytes.Buffer
	code := cli.Run([]string{"--help"}, &out, &errBuf)
	if code != report.ExitOK {
		t.Fatalf("code=%d", code)
	}
	if !strings.Contains(errBuf.String(), "Usage:") {
		t.Fatalf("usage missing: %s", errBuf.String())
	}
}
