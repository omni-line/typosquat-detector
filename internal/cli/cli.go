// Package cli parses flags and runs a scan.
package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/omni-line/typosquat-detector/internal/corpus"
	"github.com/omni-line/typosquat-detector/internal/ecosystem"
	"github.com/omni-line/typosquat-detector/internal/match"
	"github.com/omni-line/typosquat-detector/internal/report"
	"github.com/omni-line/typosquat-detector/internal/scan"
	"github.com/omni-line/typosquat-detector/internal/version"
)

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }

func (s *stringList) Set(v string) error {
	for _, part := range strings.Split(v, ",") {
		if part = strings.TrimSpace(part); part != "" {
			*s = append(*s, part)
		}
	}
	return nil
}

type config struct {
	root         string
	format       report.Format
	color        report.ColorMode
	distance     int
	failOn       scan.Severity
	strict       bool
	quiet        bool
	noMarketing  bool
	forceMarket  bool
	noScopePeers bool
	ignore       *match.Matcher
	allow        map[string]struct{}
	exclude      *match.Matcher
	scopes       []string
}

var errHelp = errors.New("help requested")

// Run executes the CLI and returns a process exit code. SIGINT/SIGTERM
// cancel the scan; a second signal terminates immediately.
func Run(args []string, stdout, stderr io.Writer) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		stop()
	}()
	return RunContext(ctx, args, stdout, stderr)
}

// RunContext is Run with a caller-supplied context.
func RunContext(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	cfg, showVersion, err := parse(args, stderr)
	switch {
	case errors.Is(err, errHelp):
		return report.ExitOK
	case err != nil:
		fmt.Fprintf(stderr, "error: %v\n", err)
		return report.ExitError
	case showVersion:
		fmt.Fprintln(stdout, version.Long())
		return report.ExitOK
	}

	start := time.Now()
	res, err := scan.Run(ctx, cfg.root, scan.Options{
		Ecosystems:   ecosystem.Default(),
		MaxDistance:  cfg.distance,
		SafeIgnore:   cfg.ignore,
		Allow:        cfg.allow,
		Exclude:      cfg.exclude,
		Scopes:       cfg.scopes,
		NoScopePeers: cfg.noScopePeers,
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			err = errors.New("interrupted")
		}
		fmt.Fprintf(stderr, "error: %v\n", err)
		return report.ExitError
	}
	elapsed := time.Since(start)

	var meta *corpus.Meta
	if m, err := corpus.MetaInfo(); err == nil {
		meta = &m
	}
	stdoutFile, _ := stdout.(*os.File)
	stderrFile, _ := stderr.(*os.File)
	return report.Write(stdout, stderr, res, report.Options{
		Format:         cfg.format,
		Version:        version.String(),
		Quiet:          cfg.quiet,
		NoMarketing:    cfg.noMarketing,
		ForceMarketing: cfg.forceMarket,
		FailOn:         cfg.failOn,
		Strict:         cfg.strict,
		StdoutIsTTY:    report.IsTTY(stdoutFile),
		StderrIsTTY:    report.IsTTY(stderrFile),
		Color:          cfg.color,
		Root:           cfg.root,
		MaxDistance:    cfg.distance,
		Duration:       elapsed,
		Corpus:         meta,
	})
}

func parse(args []string, stderr io.Writer) (cfg config, showVersion bool, err error) {
	fs := flag.NewFlagSet("typosquat-detector", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var (
		format      = fs.String("format", "text", "output format: text|json|sarif")
		distance    = fs.Int("distance", scan.DefaultDistance, "max edit distance to flag: 1|2")
		failOn      = fs.String("fail-on", "any", "exit 1 when a finding is at least: any|critical|high|medium|none")
		strict      = fs.Bool("strict", false, "exit 2 when a manifest cannot be read or parsed")
		colorMode   = fs.String("color", "auto", "color output: auto|always|never")
		quiet       = fs.Bool("q", false, "one line per finding; suppress banner, summary, and marketing")
		quietLong   = fs.Bool("quiet", false, "alias for -q")
		noMarketing = fs.Bool("no-marketing", false, "hide Omni Line CTA / JSON sponsor")
		forceMarket = fs.Bool("marketing", false, "force marketing even when non-TTY")
		noPeers     = fs.Bool("no-scope-peers", false, "disable auto intra-scope peer checks")
		versionFlag = fs.Bool("version", false, "print version and exit")
		ignore      stringList
		allow       stringList
		exclude     stringList
		scopes      stringList
	)
	fs.Var(&ignore, "ignore", "package name globs to skip (repeatable or comma-separated)")
	fs.Var(&allow, "allow", "exact package names to treat as safe (repeatable or comma-separated)")
	fs.Var(&exclude, "exclude", "path globs or directory names to skip (repeatable or comma-separated)")
	fs.Var(&scopes, "scope", "npm scopes for peer typo checks, e.g. @my-org (repeatable)")
	fs.Usage = func() { usage(fs, stderr) }

	flagArgs, positional, err := splitArgs(fs, args)
	if err != nil {
		return cfg, false, err
	}
	if err := fs.Parse(flagArgs); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return cfg, false, errHelp
		}
		return cfg, false, err
	}
	if *versionFlag {
		return cfg, true, nil
	}

	cfg.root = "."
	switch len(positional) {
	case 0:
	case 1:
		cfg.root = positional[0]
	default:
		return cfg, false, errors.New("too many path arguments (want at most one)")
	}

	if cfg.format, err = report.ParseFormat(strings.ToLower(*format)); err != nil {
		return cfg, false, err
	}
	if cfg.color, err = report.ParseColorMode(*colorMode); err != nil {
		return cfg, false, err
	}
	switch v := strings.ToLower(strings.TrimSpace(*failOn)); v {
	case "any":
		cfg.failOn = scan.SeverityMedium
	case "none":
	default:
		if cfg.failOn, err = scan.ParseSeverity(v); err != nil {
			return cfg, false, fmt.Errorf("invalid --fail-on %q (want any|critical|high|medium|none)", *failOn)
		}
	}
	if *distance != 1 && *distance != 2 {
		return cfg, false, fmt.Errorf("invalid --distance %d (want 1|2)", *distance)
	}
	cfg.distance = *distance
	cfg.strict = *strict
	cfg.ignore = match.New(ignore...)
	cfg.exclude = match.New(exclude...)
	cfg.allow = map[string]struct{}{}
	for _, a := range allow {
		cfg.allow[a] = struct{}{}
	}
	cfg.scopes = scopes
	cfg.quiet = *quiet || *quietLong
	cfg.noMarketing = *noMarketing || envTruthy("TYPOSQUAT_NO_MARKETING") || envTruthy("OMNI_AUDIT_NO_MARKETING")
	cfg.forceMarket = *forceMarket
	cfg.noScopePeers = *noPeers
	return cfg, false, nil
}

func usage(fs *flag.FlagSet, w io.Writer) {
	fmt.Fprint(w, `Usage: typosquat-detector [path] [flags]

Scan a project tree for npm and PyPI dependency names that are 1–2 edits away
from a high-download public package (typosquatting). Works fully offline using
an embedded popular-package corpus.

Flags:
`)
	fs.PrintDefaults()
	fmt.Fprint(w, `
Severities:
  critical  1 edit from a popular package
  high      2 edits from a popular package, or 1 edit from a scope peer
  medium    2 edits from a scope peer

Exit codes:
  0  no findings at or above --fail-on (or --fail-on none)
  1  findings at or above --fail-on
  2  usage or runtime error, or unreadable manifests with --strict

Environment:
  TYPOSQUAT_NO_MARKETING=1   same as --no-marketing
  NO_COLOR=1                 disable ANSI colors (also --color never)
  FORCE_COLOR=1              enable colors even when non-TTY
`)
}

// splitArgs lets flags appear after the positional path, which the standard
// flag package does not support.
func splitArgs(fs *flag.FlagSet, args []string) (flagArgs, positional []string, err error) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		if len(a) < 2 || a[0] != '-' {
			positional = append(positional, a)
			continue
		}
		flagArgs = append(flagArgs, a)
		name := strings.TrimLeft(a, "-")
		if strings.Contains(name, "=") {
			continue
		}
		f := fs.Lookup(name)
		if f == nil {
			continue
		}
		if bf, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && bf.IsBoolFlag() {
			continue
		}
		if i+1 >= len(args) {
			return nil, nil, fmt.Errorf("flag %s requires a value", a)
		}
		i++
		flagArgs = append(flagArgs, args[i])
	}
	return flagArgs, positional, nil
}

func envTruthy(key string) bool {
	switch strings.TrimSpace(strings.ToLower(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
