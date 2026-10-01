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
	failAny      bool
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

// Run executes the CLI and returns a process exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	_ = ctx

	cfg, showVersion, err := parse(args, stderr)
	switch {
	case errors.Is(err, errHelp):
		return report.ExitOK
	case err != nil:
		fmt.Fprintf(stderr, "error: %v\n", err)
		return report.ExitError
	case showVersion:
		fmt.Fprintln(stdout, version.String())
		return report.ExitOK
	}

	res, err := scan.Run(cfg.root, scan.Options{
		Ecosystems:   ecosystem.Default(),
		MaxDistance:  cfg.distance,
		SafeIgnore:   cfg.ignore,
		Allow:        cfg.allow,
		Exclude:      cfg.exclude,
		Scopes:       cfg.scopes,
		NoScopePeers: cfg.noScopePeers,
	})
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return report.ExitError
	}

	stdoutFile, _ := stdout.(*os.File)
	stderrFile, _ := stderr.(*os.File)
	return report.Write(stdout, stderr, res, report.Options{
		Format:      cfg.format,
		Version:     version.String(),
		Quiet:       cfg.quiet,
		NoMarketing: cfg.noMarketing,
		ForceMarket: cfg.forceMarket,
		FailOnAny:   cfg.failAny,
		StdoutIsTTY: report.IsTTY(stdoutFile),
		StderrIsTTY: report.IsTTY(stderrFile),
		StdoutPiped: !report.IsTTY(stdoutFile),
		Color:       cfg.color,
	})
}

func parse(args []string, stderr io.Writer) (cfg config, showVersion bool, err error) {
	fs := flag.NewFlagSet("typosquat-detector", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var (
		format      = fs.String("format", "text", "output format: text|json")
		distance    = fs.Int("distance", scan.DefaultDistance, "max edit distance to flag: 1|2")
		failOn      = fs.String("fail-on", "any", "when to exit 1: any|none")
		colorMode   = fs.String("color", "auto", "color output: auto|always|never")
		quiet       = fs.Bool("q", false, "findings only; suppress banner, summary, and marketing")
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

	switch strings.ToLower(*format) {
	case "text":
		cfg.format = report.FormatText
	case "json":
		cfg.format = report.FormatJSON
	default:
		return cfg, false, fmt.Errorf("invalid --format %q (want text|json)", *format)
	}
	if cfg.color, err = report.ParseColorMode(*colorMode); err != nil {
		return cfg, false, err
	}
	switch strings.ToLower(*failOn) {
	case "any":
		cfg.failAny = true
	case "none":
	default:
		return cfg, false, fmt.Errorf("invalid --fail-on %q (want any|none)", *failOn)
	}
	if *distance != 1 && *distance != 2 {
		return cfg, false, fmt.Errorf("invalid --distance %d (want 1|2)", *distance)
	}
	cfg.distance = *distance
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
Exit codes:
  0  no findings (or --fail-on none)
  1  findings reported
  2  usage or runtime error

Environment:
  TYPOSQUAT_NO_MARKETING=1   same as --no-marketing
  NO_COLOR=1                 disable ANSI colors (also --color never)
  FORCE_COLOR=1              enable colors even when non-TTY
`)
}

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
