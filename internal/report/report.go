// Package report renders scan results as text, JSON or SARIF and maps them
// to process exit codes.
package report

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/omni-line/typosquat-detector/internal/corpus"
	"github.com/omni-line/typosquat-detector/internal/scan"
)

// Process exit codes.
const (
	ExitOK       = 0
	ExitFindings = 1
	ExitError    = 2
)

// Format selects the output renderer.
type Format string

// Supported formats.
const (
	FormatText  Format = "text"
	FormatJSON  Format = "json"
	FormatSARIF Format = "sarif"
)

// ParseFormat parses a --format value.
func ParseFormat(v string) (Format, error) {
	switch f := Format(v); f {
	case FormatText, FormatJSON, FormatSARIF:
		return f, nil
	default:
		return "", fmt.Errorf("invalid --format %q (want text|json|sarif)", v)
	}
}

// Options controls rendering and exit status.
type Options struct {
	Format         Format
	Version        string
	Quiet          bool
	NoMarketing    bool
	ForceMarketing bool
	// FailOn is the minimum severity that makes the process exit 1.
	// Empty means findings never fail the process.
	FailOn scan.Severity
	// Strict makes warnings (unreadable or unparsable manifests) exit 2 when
	// there are no failing findings.
	Strict      bool
	StdoutIsTTY bool
	StderrIsTTY bool
	Color       ColorMode

	// Scan context, included in rich output when set.
	Root        string
	MaxDistance int
	Duration    time.Duration
	Corpus      *corpus.Meta
}

// Write renders res and returns the process exit code.
func Write(stdout, stderr io.Writer, res *scan.Result, opts Options) int {
	if res == nil {
		fmt.Fprintln(stderr, "error: empty scan result")
		return ExitError
	}
	var err error
	switch opts.Format {
	case FormatJSON:
		err = writeJSON(stdout, res, opts)
	case FormatSARIF:
		err = writeSARIF(stdout, res, opts)
	default:
		writeText(stdout, stderr, res, opts)
	}
	if err != nil {
		fmt.Fprintf(stderr, "error: write %s: %v\n", opts.Format, err)
		return ExitError
	}
	return ExitCode(res, opts)
}

// ExitCode maps a result to an exit code under opts.
func ExitCode(res *scan.Result, opts Options) int {
	if opts.FailOn != "" {
		for _, f := range res.Findings {
			if f.Severity.AtLeast(opts.FailOn) {
				return ExitFindings
			}
		}
	}
	if opts.Strict && len(res.Warnings) > 0 {
		return ExitError
	}
	return ExitOK
}

func shouldShowMarketing(opts Options) bool {
	if opts.Quiet || opts.NoMarketing || opts.Format == FormatSARIF {
		return false
	}
	if opts.ForceMarketing || opts.Format == FormatJSON {
		return true
	}
	return opts.StdoutIsTTY || opts.StderrIsTTY
}

// IsTTY reports whether f is a terminal.
func IsTTY(f *os.File) bool {
	if f == nil {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
