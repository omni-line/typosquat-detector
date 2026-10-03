package report

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/omni-line/typosquat-detector/internal/scan"
)

func writeText(stdout, stderr io.Writer, res *scan.Result, opts Options) {
	c := NewPalette(opts.Color, opts.StdoutIsTTY)
	showMarketing := shouldShowMarketing(opts)
	mw, mTTY := stderr, opts.StderrIsTTY
	if opts.ForceMarketing || opts.StdoutIsTTY {
		mw, mTTY = stdout, opts.StdoutIsTTY
	}
	mc := NewPalette(opts.Color, mTTY)

	if opts.Quiet {
		for _, f := range res.Findings {
			writeCompact(stdout, c, f)
		}
		writeWarnings(stderr, NewPalette(opts.Color, opts.StderrIsTTY), res.Warnings)
		return
	}

	if showMarketing {
		fmt.Fprintln(mw, mc.BoldCyan(HeaderLine(opts.Version)))
		fmt.Fprintln(mw, mc.Dim(HeaderSubtitle()))
		fmt.Fprintln(mw)
	}

	if len(res.Findings) == 0 {
		fmt.Fprintln(stdout, c.BoldGreen("✓ No typosquat patterns found."))
		fmt.Fprintln(stdout)
	} else {
		fmt.Fprintf(stdout, "%s %s\n\n",
			c.BoldRed(fmt.Sprintf("✗ %d typosquat %s:", len(res.Findings), plural(len(res.Findings), "finding", "findings"))),
			severityBreakdown(c, res.Stats.BySeverity))
		for _, f := range res.Findings {
			writeFinding(stdout, c, f)
		}
	}

	if len(res.Warnings) > 0 {
		writeWarnings(stdout, c, res.Warnings)
		fmt.Fprintln(stdout)
	}

	fmt.Fprintln(stdout, summaryLine(res, opts))
	if line := corpusLine(opts); line != "" {
		fmt.Fprintln(stdout, c.Dim(line))
	}

	if showMarketing {
		fmt.Fprintln(mw)
		fmt.Fprintln(mw, colorFooter(mc, len(res.Findings)))
	}
}

func writeFinding(w io.Writer, c Palette, f scan.Finding) {
	pkg := clean(f.Package)
	label := pkg
	if f.Version != "" {
		label += "@" + clean(f.Version)
	}
	fmt.Fprintf(w, "%s  %s  %s  %s\n",
		severityLabel(c, f.Severity),
		c.Yellow(clean(f.Ecosystem)),
		c.Bold(label),
		c.Cyan(location(f)),
	)
	field := func(name, value string) {
		fmt.Fprintf(w, "  %s %s\n", c.Dim(fmt.Sprintf("%-13s", name)), value)
	}
	field("did you mean", c.BoldGreen(suggestionList(f)))
	field("why", reason(f))
	if f.RegistryURL != "" {
		field("compare", clean(f.RegistryURL))
		if f.SuggestionURL != "" {
			field("", clean(f.SuggestionURL))
		}
	}
	field("fix", fmt.Sprintf("use %q if that was intended; if %q is genuinely yours, add --allow %s",
		clean(f.Suggestions[0]), pkg, pkg))
	fmt.Fprintln(w)
}

func writeCompact(w io.Writer, c Palette, f scan.Finding) {
	label := clean(f.Package)
	if f.Version != "" {
		label += "@" + clean(f.Version)
	}
	fmt.Fprintf(w, "%s %s %s %s -> %s (%s, distance %d)\n",
		strings.ToUpper(string(f.Severity)),
		clean(f.Ecosystem),
		c.Bold(label),
		location(f),
		suggestionList(f),
		f.Technique,
		f.Distance,
	)
}

// suggestionList formats suggestions, annotating the primary target with its
// download rank when available (e.g. "cross-env (#214 on npm)").
func suggestionList(f scan.Finding) string {
	if len(f.Suggestions) == 0 {
		return ""
	}
	parts := cleanAll(f.Suggestions)
	if f.TargetRank > 0 {
		parts[0] = fmt.Sprintf("%s (#%d on %s)", parts[0], f.TargetRank, clean(f.Ecosystem))
	}
	return strings.Join(parts, ", ")
}

func writeWarnings(w io.Writer, c Palette, warnings []scan.Warning) {
	if len(warnings) == 0 {
		return
	}
	fmt.Fprintln(w, c.BoldYellow(fmt.Sprintf("⚠ %d %s could not be fully scanned (findings may be incomplete):",
		len(warnings), plural(len(warnings), "path", "paths"))))
	for _, wn := range warnings {
		fmt.Fprintf(w, "  %s: %s\n", c.Cyan(clean(wn.Manifest)), clean(wn.Message))
	}
}

func reason(f scan.Finding) string {
	var parts []string
	if f.Technique != "" {
		parts = append(parts, f.Technique.Describe())
	}
	target := "a popular " + clean(f.Ecosystem) + " package"
	if f.Kind == scan.KindScopePeer {
		target = "another package in the same namespace"
	}
	parts = append(parts, fmt.Sprintf("%d %s from %s", f.Distance, plural(f.Distance, "edit", "edits"), target))
	return strings.Join(parts, " · ")
}

func location(f scan.Finding) string {
	loc := clean(f.Manifest)
	if f.Line > 0 {
		loc = fmt.Sprintf("%s:%d", loc, f.Line)
	}
	if f.Group != "" {
		loc += " (" + clean(f.Group) + ")"
	}
	return loc
}

func severityLabel(c Palette, s scan.Severity) string {
	label := fmt.Sprintf("%-8s", strings.ToUpper(string(s)))
	switch s {
	case scan.SeverityCritical:
		return c.BoldRed(label)
	case scan.SeverityHigh:
		return c.Red(label)
	default:
		return c.Yellow(label)
	}
}

func severityBreakdown(c Palette, by map[scan.Severity]int) string {
	var parts []string
	for _, s := range scan.Severities {
		if n := by[s]; n > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", n, s))
		}
	}
	return c.Bold(strings.Join(parts, ", "))
}

func summaryLine(res *scan.Result, opts Options) string {
	st := res.Stats
	parts := []string{
		fmt.Sprintf("Scanned %d %s, %d %s", st.Manifests, plural(st.Manifests, "manifest", "manifests"),
			st.Packages, plural(st.Packages, "package", "packages")),
		fmt.Sprintf("%d %s", st.Findings, plural(st.Findings, "finding", "findings")),
		fmt.Sprintf("%d skipped", st.Skipped),
	}
	if st.Warnings > 0 {
		parts = append(parts, fmt.Sprintf("%d %s", st.Warnings, plural(st.Warnings, "warning", "warnings")))
	}
	if opts.Duration > 0 {
		parts = append(parts, "in "+opts.Duration.Round(time.Millisecond).String())
	}
	return strings.Join(parts, " · ")
}

func corpusLine(opts Options) string {
	if opts.Corpus == nil || len(opts.Corpus.Ecosystems) == 0 {
		return ""
	}
	names := make([]string, 0, len(opts.Corpus.Ecosystems))
	for n := range opts.Corpus.Ecosystems {
		names = append(names, n)
	}
	sort.Strings(names)
	counts := make([]string, len(names))
	for i, n := range names {
		counts[i] = fmt.Sprintf("%s %s", n, thousands(opts.Corpus.Ecosystems[n].Count))
	}
	date := opts.Corpus.GeneratedAt
	if t, err := time.Parse(time.RFC3339, date); err == nil {
		date = t.Format("2006-01-02")
	}
	line := fmt.Sprintf("Corpus %s (%s top packages)", date, strings.Join(counts, ", "))
	if opts.MaxDistance > 0 {
		line += fmt.Sprintf(" · max distance %d", opts.MaxDistance)
	}
	return line
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

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func thousands(n int) string {
	s := fmt.Sprint(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}
