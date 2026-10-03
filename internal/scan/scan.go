// Package scan discovers manifests and compares dependency names against
// popular-package corpora and against namespace peers.
package scan

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/omni-line/typosquat-detector/internal/corpus"
	"github.com/omni-line/typosquat-detector/internal/discover"
	"github.com/omni-line/typosquat-detector/internal/distance"
	"github.com/omni-line/typosquat-detector/internal/ecosystem"
	"github.com/omni-line/typosquat-detector/internal/manifest"
	"github.com/omni-line/typosquat-detector/internal/match"
)

// DefaultDistance is the default maximum edit distance flagged.
const DefaultDistance = 2

// maxSuggestions caps how many equally-close targets a finding lists.
const maxSuggestions = 3

// Finding is a suspected typosquat at one manifest location.
type Finding struct {
	Ecosystem   string   `json:"ecosystem"`
	Package     string   `json:"package"`
	Version     string   `json:"version,omitempty"`
	Manifest    string   `json:"manifest"`
	Line        int      `json:"line,omitempty"`
	Group       string   `json:"group,omitempty"`
	Distance    int      `json:"distance"`
	Suggestions []string `json:"suggestions"`
	// TargetRank is the 1-based download rank of Suggestions[0] when the
	// corpus is rank-ordered; omitted when unavailable.
	TargetRank    int                `json:"target_rank,omitempty"`
	Kind          Kind               `json:"kind"`
	Severity      Severity           `json:"severity"`
	Technique     distance.Technique `json:"technique,omitempty"`
	Message       string             `json:"message,omitempty"`
	PURL          string             `json:"purl,omitempty"`
	RegistryURL   string             `json:"registry_url,omitempty"`
	SuggestionURL string             `json:"suggestion_url,omitempty"`
}

// Warning is a non-fatal problem that may hide findings, such as a manifest
// that could not be read or parsed.
type Warning struct {
	Manifest string `json:"manifest"`
	Message  string `json:"message"`
}

// Stats summarizes a scan.
type Stats struct {
	Manifests  int              `json:"manifests"`
	Packages   int              `json:"packages"`
	Findings   int              `json:"findings"`
	Skipped    int              `json:"skipped"`
	Warnings   int              `json:"warnings"`
	BySeverity map[Severity]int `json:"by_severity"`
}

// Result is the full scan outcome.
type Result struct {
	Findings []Finding `json:"findings"`
	Warnings []Warning `json:"warnings"`
	Stats    Stats     `json:"stats"`
}

// Options configures Run.
type Options struct {
	Ecosystems  []ecosystem.Ecosystem
	MaxDistance int
	// SafeIgnore skips package names matching these globs.
	SafeIgnore *match.Matcher
	// Allow lists exact (or normalized) package names treated as safe.
	Allow   map[string]struct{}
	Exclude *match.Matcher
	// Scopes are namespaces (npm "@org") to peer-check in addition to the
	// ones discovered automatically.
	Scopes       []string
	NoScopePeers bool
}

type decl struct {
	eco *ecosystem.Ecosystem
	dep manifest.Dependency
	key string
	rel string
}

type popularHit struct {
	dist    int
	targets []string
}

type scanner struct {
	opts     Options
	corpora  map[string]*corpus.Set
	popular  map[string]popularHit
	res      *Result
	reported map[string]int
}

// Run discovers manifests under root and reports suspected typosquats.
func Run(ctx context.Context, root string, opts Options) (*Result, error) {
	if opts.MaxDistance <= 0 {
		opts.MaxDistance = DefaultDistance
	}
	if opts.MaxDistance > 2 {
		opts.MaxDistance = 2
	}
	if err := ecosystem.ValidateAll(opts.Ecosystems); err != nil {
		return nil, err
	}
	root, err := resolveRoot(root)
	if err != nil {
		return nil, err
	}

	s := &scanner{
		opts:     opts,
		corpora:  map[string]*corpus.Set{},
		popular:  map[string]popularHit{},
		res:      &Result{Findings: []Finding{}, Warnings: []Warning{}},
		reported: map[string]int{},
	}
	byName := map[string]*ecosystem.Ecosystem{}
	for i := range opts.Ecosystems {
		e := &opts.Ecosystems[i]
		byName[e.Name] = e
		set, err := e.LoadCorpus()
		if err != nil {
			return nil, fmt.Errorf("ecosystem %s: %w", e.Name, err)
		}
		s.corpora[e.Name] = set
	}

	manifests, err := discover.Walk(ctx, root, discover.Options{
		Exclude: opts.Exclude,
		Classify: func(rel string) string {
			for _, e := range opts.Ecosystems {
				if e.IsManifest(rel) {
					return e.Name
				}
			}
			return ""
		},
		Warn: func(rel string, err error) { s.warn(rel, err) },
	})
	if err != nil {
		return nil, err
	}
	s.res.Stats.Manifests = len(manifests)

	var decls []decl
	for _, m := range manifests {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		eco := byName[m.Ecosystem]
		data, err := manifest.ReadFile(m.Path)
		if err != nil {
			s.warn(m.Rel, err)
			continue
		}
		deps, err := eco.Parse(m.Rel, data)
		if err != nil {
			s.warn(m.Rel, err)
			continue
		}
		for _, d := range deps {
			s.res.Stats.Packages++
			key := eco.Key(d.Name)
			if s.allowed(d.Name, key) {
				s.res.Stats.Skipped++
				continue
			}
			decls = append(decls, decl{eco: eco, dep: d, key: key, rel: m.Rel})
		}
	}

	for _, d := range decls {
		s.checkPopular(d)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.checkPeers(decls)

	s.finish()
	return s.res, nil
}

func resolveRoot(root string) (string, error) {
	fi, err := os.Lstat(root)
	if err != nil {
		return "", err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return filepath.EvalSymlinks(root)
	}
	return root, nil
}

func (s *scanner) warn(rel string, err error) {
	s.res.Warnings = append(s.res.Warnings, Warning{Manifest: rel, Message: err.Error()})
}

func (s *scanner) allowed(name, key string) bool {
	if s.opts.SafeIgnore.Match(name) || s.opts.SafeIgnore.Match(key) {
		return true
	}
	if _, ok := s.opts.Allow[name]; ok {
		return true
	}
	_, ok := s.opts.Allow[key]
	return ok
}

func (s *scanner) checkPopular(d decl) {
	set := s.corpora[d.eco.Name]
	if set.Contains(d.key) || (d.eco.Implied != nil && d.eco.Implied(d.key, set.Contains)) {
		return
	}
	memo := d.eco.Name + "\x00" + d.key
	hit, ok := s.popular[memo]
	if !ok {
		hit = nearest(set, d.key, s.opts.MaxDistance)
		s.popular[memo] = hit
	}
	if len(hit.targets) == 0 {
		return
	}
	display := make([]string, len(hit.targets))
	for i, t := range hit.targets {
		display[i] = set.Display(t)
	}
	var targetRank int
	if r, ok := set.Rank(hit.targets[0]); ok {
		targetRank = r
	}
	s.add(d, KindPopular, hit.dist, distance.Classify(d.key, hit.targets[0]), display, targetRank)
}

// nearest returns the closest corpus keys to key within maxDist.
func nearest(set *corpus.Set, key string, maxDist int) popularHit {
	best := popularHit{dist: maxDist + 1}
	seen := map[string]struct{}{}
	set.Candidates(key, maxDist, func(c string) {
		if _, dup := seen[c]; dup {
			return
		}
		seen[c] = struct{}{}
		limit := budget(key, c, maxDist)
		if limit == 0 {
			return
		}
		d, ok := distance.Within(key, c, limit)
		if !ok || d == 0 {
			return
		}
		if d < best.dist {
			best.dist = d
			best.targets = best.targets[:0]
		}
		if d == best.dist {
			best.targets = append(best.targets, c)
		}
	})
	sort.Slice(best.targets, func(i, j int) bool {
		ri, oki := set.Rank(best.targets[i])
		rj, okj := set.Rank(best.targets[j])
		switch {
		case oki && okj && ri != rj:
			return ri < rj
		case oki != okj:
			return oki
		default:
			return best.targets[i] < best.targets[j]
		}
	})
	if len(best.targets) > maxSuggestions {
		best.targets = best.targets[:maxSuggestions]
	}
	return best
}

// budget caps the distance considered for short names: two edits can turn
// almost any 3–4 letter name into another real package, which would drown
// real findings in noise.
func budget(a, b string, maxDist int) int {
	n := len(distance.StripSeparators(a))
	if m := len(distance.StripSeparators(b)); m < n {
		n = m
	}
	switch {
	case n < 3:
		return 0
	case n < 5 && maxDist > 1:
		return 1
	default:
		return maxDist
	}
}

type peerHit struct {
	dist  int
	goods []string
}

// checkPeers flags packages in the same namespace that are near-misses of
// each other (e.g. @acme/authh next to @acme/auth).
func (s *scanner) checkPeers(decls []decl) {
	wanted := map[string]struct{}{}
	for _, sc := range s.opts.Scopes {
		if sc = strings.TrimSpace(sc); sc != "" {
			wanted[strings.ToLower(sc)] = struct{}{}
		}
	}
	type group struct {
		eco   *ecosystem.Ecosystem
		names map[string][]decl
	}
	groups := map[string]*group{}
	for _, d := range decls {
		if d.eco.Namespace == nil {
			continue
		}
		ns, _ := d.eco.Namespace(d.key)
		if ns == "" {
			continue
		}
		if _, ok := wanted[ns]; !ok && s.opts.NoScopePeers {
			continue
		}
		gk := d.eco.Name + "\x00" + ns
		g := groups[gk]
		if g == nil {
			g = &group{eco: d.eco, names: map[string][]decl{}}
			groups[gk] = g
		}
		g.names[d.key] = append(g.names[d.key], d)
	}

	for _, g := range groups {
		names := make([]string, 0, len(g.names))
		for n := range g.names {
			names = append(names, n)
		}
		sort.Strings(names)
		set := s.corpora[g.eco.Name]
		hits := map[string]*peerHit{}
		for i := 0; i < len(names); i++ {
			for j := i + 1; j < len(names); j++ {
				a, b := names[i], names[j]
				_, aLeaf := g.eco.Namespace(a)
				_, bLeaf := g.eco.Namespace(b)
				if isSiblingFamily(aLeaf, bLeaf) {
					continue
				}
				limit := budget(aLeaf, bLeaf, s.opts.MaxDistance)
				if limit == 0 {
					continue
				}
				d, ok := distance.Within(aLeaf, bLeaf, limit)
				if !ok || d == 0 {
					continue
				}
				typo, good := pickTypo(a, aLeaf, b, bLeaf, set)
				h := hits[typo]
				if h == nil || d < h.dist {
					h = &peerHit{dist: d}
					hits[typo] = h
				}
				if d == h.dist && len(h.goods) < maxSuggestions {
					h.goods = append(h.goods, good)
				}
			}
		}
		for typo, h := range hits {
			_, tLeaf := g.eco.Namespace(typo)
			_, gLeaf := g.eco.Namespace(h.goods[0])
			tech := distance.Classify(tLeaf, gLeaf)
			for _, d := range g.names[typo] {
				s.add(d, KindScopePeer, h.dist, tech, h.goods, 0)
			}
		}
	}
}

// isSiblingFamily reports whether one leaf extends the other by a whole
// separator-delimited segment (swagger / swagger-ui, ui / ui-kit). That is how
// organizations name related packages, not how typos look.
func isSiblingFamily(a, b string) bool {
	if len(a) > len(b) {
		a, b = b, a
	}
	if a == "" || len(b) < len(a)+2 {
		return false
	}
	isSep := func(c byte) bool { return c == '-' || c == '_' || c == '.' }
	return (strings.HasPrefix(b, a) && isSep(b[len(a)])) ||
		(strings.HasSuffix(b, a) && isSep(b[len(b)-len(a)-1]))
}

// pickTypo decides which of two near-identical peers is the likely mistake:
// a name in the popular corpus wins, then the shorter leaf, then lexical order.
func pickTypo(a, aLeaf, b, bLeaf string, set *corpus.Set) (typo, good string) {
	aPop, bPop := set.Contains(a), set.Contains(b)
	switch {
	case aPop && !bPop:
		return b, a
	case bPop && !aPop:
		return a, b
	case len(aLeaf) != len(bLeaf):
		if len(aLeaf) > len(bLeaf) {
			return a, b
		}
		return b, a
	case a > b:
		return a, b
	default:
		return b, a
	}
}

func (s *scanner) add(d decl, kind Kind, dist int, tech distance.Technique, targets []string, targetRank int) {
	key := strings.Join([]string{d.eco.Name, d.rel, d.key, string(kind)}, "\x00")
	if _, dup := s.reported[key]; dup {
		return
	}
	s.reported[key] = len(s.res.Findings)
	f := Finding{
		Ecosystem:   d.eco.Name,
		Package:     d.dep.Name,
		Version:     d.dep.Version,
		Manifest:    d.rel,
		Line:        d.dep.Line,
		Group:       d.dep.Group,
		Distance:    dist,
		Suggestions: targets,
		TargetRank:  targetRank,
		Kind:        kind,
		Severity:    severityFor(kind, dist),
		Technique:   tech,
		PURL:        d.eco.PackageID(d.dep.Name),
		RegistryURL: d.eco.URL(d.dep.Name),
	}
	if len(targets) > 0 {
		f.SuggestionURL = d.eco.URL(targets[0])
	}
	f.Message = message(f)
	s.res.Findings = append(s.res.Findings, f)
}

func message(f Finding) string {
	what := "a popular " + f.Ecosystem + " package"
	if f.Kind == KindScopePeer {
		what = "a package in the same namespace"
	}
	edits := "edit"
	if f.Distance != 1 {
		edits = "edits"
	}
	msg := fmt.Sprintf("%q is %d %s away from %s, %q", f.Package, f.Distance, edits, what, f.Suggestions[0])
	if f.Technique != "" {
		msg += " (" + f.Technique.Describe() + ")"
	}
	return msg
}

func (s *scanner) finish() {
	r := s.res
	sort.SliceStable(r.Findings, func(i, j int) bool {
		a, b := r.Findings[i], r.Findings[j]
		if a.Severity.Rank() != b.Severity.Rank() {
			return a.Severity.Rank() > b.Severity.Rank()
		}
		if a.Ecosystem != b.Ecosystem {
			return a.Ecosystem < b.Ecosystem
		}
		if a.Manifest != b.Manifest {
			return a.Manifest < b.Manifest
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Package < b.Package
	})
	r.Stats.Findings = len(r.Findings)
	r.Stats.Warnings = len(r.Warnings)
	r.Stats.BySeverity = map[Severity]int{}
	for _, sev := range Severities {
		r.Stats.BySeverity[sev] = 0
	}
	for _, f := range r.Findings {
		r.Stats.BySeverity[f.Severity]++
	}
}
