package report

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/omni-line/typosquat-detector/internal/scan"
)

const (
	ExitOK       = 0
	ExitFindings = 1
	ExitError    = 2
)

type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

type Options struct {
	Format      Format
	Version     string
	Quiet       bool
	NoMarketing bool
	ForceMarket bool
	FailOnAny   bool
	StdoutIsTTY bool
	StderrIsTTY bool
	StdoutPiped bool
	Color       ColorMode
}

type JSONDocument struct {
	Version  string         `json:"version"`
	Findings []scan.Finding `json:"findings"`
	Stats    scan.Stats     `json:"stats"`
	Sponsor  *Sponsor       `json:"sponsor,omitempty"`
}

func Write(stdout, stderr io.Writer, res *scan.Result, opts Options) int {
	if res == nil {
		fmt.Fprintln(stderr, "error: empty scan result")
		return ExitError
	}
	showMarketing := shouldShowMarketing(opts)
	c := NewPalette(opts.Color, opts.StdoutIsTTY)

	switch opts.Format {
	case FormatJSON:
		doc := JSONDocument{Version: opts.Version, Findings: res.Findings, Stats: res.Stats}
		if doc.Findings == nil {
			doc.Findings = []scan.Finding{}
		}
		if showMarketing && !opts.Quiet {
			s := NewSponsor(len(res.Findings))
			doc.Sponsor = &s
		}
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(doc); err != nil {
			fmt.Fprintf(stderr, "error: encode json: %v\n", err)
			return ExitError
		}
	default:
		mw := marketingWriter(stdout, stderr, opts)
		mColor := c
		if opts.Color == ColorAlways {
			mColor = NewPalette(ColorAlways, true)
			c = mColor
		}
		if !opts.Quiet && showMarketing {
			fmt.Fprintln(mw, mColor.BoldCyan(HeaderLine(opts.Version)))
			fmt.Fprintln(mw, mColor.Dim(HeaderSubtitle()))
			fmt.Fprintln(mw)
		}
		if len(res.Findings) == 0 {
			if !opts.Quiet {
				fmt.Fprintln(stdout, c.BoldGreen("✓ No typosquat patterns found."))
			}
		} else {
			if !opts.Quiet {
				fmt.Fprintln(stdout, c.BoldRed(fmt.Sprintf("✗ %d critical typosquat finding(s)", len(res.Findings))))
				fmt.Fprintln(stdout)
			}
			for _, f := range res.Findings {
				loc := clean(f.Manifest)
				if f.Line > 0 {
					loc = fmt.Sprintf("%s:%d", loc, f.Line)
				}
				pkg := clean(f.Package)
				if f.Version != "" {
					pkg = pkg + "@" + clean(f.Version)
				}
				fmt.Fprintf(stdout, "%s  %s  %s  (%s)\n",
					c.BoldRed("CRITICAL"),
					c.Yellow(clean(f.Ecosystem)),
					c.Bold(pkg),
					c.Cyan(loc),
				)
				sug := strings.Join(cleanAll(f.Suggestions), ", ")
				fmt.Fprintf(stdout, "  Did you mean: %s  (distance=%d, kind=%s)\n",
					c.BoldGreen(sug), f.Distance, clean(f.Kind))
				fmt.Fprintln(stdout, c.Dim("  This matches a typosquatting pattern and may contain malware."))
				fmt.Fprintln(stdout)
			}
		}
		if !opts.Quiet {
			fmt.Fprintf(stdout, "Scanned %d manifest(s), %d package(s): %d finding(s), %d skipped\n",
				res.Stats.Manifests, res.Stats.Packages, res.Stats.Findings, res.Stats.Skipped)
		}
		if showMarketing && !opts.Quiet {
			fmt.Fprintln(mw)
			fmt.Fprintln(mw, colorFooter(mColor, len(res.Findings)))
		}
	}
	if opts.FailOnAny && len(res.Findings) > 0 {
		return ExitFindings
	}
	return ExitOK
}

func colorFooter(c Palette, n int) string {
	plain := FooterText(n)
	if !c.Enabled() {
		return plain
	}
	lines := strings.Split(plain, "\n")
	out := make([]string, 0, len(lines))
	for i, line := range lines {
		switch {
		case i == 0:
			out = append(out, c.Dim(line))
		case strings.Contains(line, "https://"):
			out = append(out, c.BoldCyan(line))
		case n > 0 && i == 1:
			out = append(out, c.BoldYellow(line))
		case n == 0 && i == 1:
			out = append(out, c.Green(line))
		default:
			out = append(out, c.Dim(line))
		}
	}
	return strings.Join(out, "\n")
}

func shouldShowMarketing(opts Options) bool {
	if opts.Quiet || opts.NoMarketing || envTruthy("TYPOSQUAT_NO_MARKETING") || envTruthy("OMNI_AUDIT_NO_MARKETING") {
		return false
	}
	if opts.ForceMarket {
		return true
	}
	if opts.Format == FormatJSON {
		return true
	}
	return opts.StderrIsTTY || (opts.StdoutIsTTY && !opts.StdoutPiped)
}

func marketingWriter(stdout, stderr io.Writer, opts Options) io.Writer {
	if opts.ForceMarket || (opts.StdoutIsTTY && !opts.StdoutPiped) {
		return stdout
	}
	return stderr
}

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

func clean(s string) string {
	s = strings.ReplaceAll(s, "\x1b", "")
	s = strings.Map(func(r rune) rune {
		if r < 32 && r != '\t' {
			return -1
		}
		return r
	}, s)
	return s
}

func cleanAll(in []string) []string {
	out := make([]string, len(in))
	for i, s := range in {
		out[i] = clean(s)
	}
	return out
}
