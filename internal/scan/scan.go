package scan

import (
	"fmt"
	"sort"
	"strings"

	"github.com/omni-line/typosquat-detector/internal/discover"
	"github.com/omni-line/typosquat-detector/internal/distance"
	"github.com/omni-line/typosquat-detector/internal/ecosystem"
	"github.com/omni-line/typosquat-detector/internal/manifest"
	"github.com/omni-line/typosquat-detector/internal/manifest/npm"
	"github.com/omni-line/typosquat-detector/internal/match"
)

const DefaultDistance = 2

// Finding is a suspected typosquat.
type Finding struct {
	Ecosystem   string   `json:"ecosystem"`
	Package     string   `json:"package"`
	Version     string   `json:"version,omitempty"`
	Manifest    string   `json:"manifest"`
	Line        int      `json:"line,omitempty"`
	Group       string   `json:"group,omitempty"`
	Distance    int      `json:"distance"`
	Suggestions []string `json:"suggestions"`
	Kind        string   `json:"kind"` // popular | scope-peer
	Severity    string   `json:"severity"`
}

// Stats summarizes a scan.
type Stats struct {
	Manifests int `json:"manifests"`
	Packages  int `json:"packages"`
	Findings  int `json:"findings"`
	Skipped   int `json:"skipped"`
}

// Result is the full scan outcome.
type Result struct {
	Findings []Finding `json:"findings"`
	Stats    Stats     `json:"stats"`
}

// Options configures Run.
type Options struct {
	Ecosystems   []ecosystem.Ecosystem
	MaxDistance  int
	SafeIgnore   *match.Matcher
	Allow        map[string]struct{}
	Exclude      *match.Matcher
	Scopes       []string
	NoScopePeers bool
}

// Run discovers manifests and compares dependency names to popular corpora.
func Run(root string, opts Options) (*Result, error) {
	if opts.MaxDistance <= 0 {
		opts.MaxDistance = DefaultDistance
	}
	if opts.MaxDistance > 2 {
		opts.MaxDistance = 2
	}
	if err := ecosystem.ValidateAll(opts.Ecosystems); err != nil {
		return nil, err
	}
	byName := map[string]*ecosystem.Ecosystem{}
	for i := range opts.Ecosystems {
		e := &opts.Ecosystems[i]
		byName[e.Name] = e
	}

	manifests, err := discover.Walk(root, discover.Options{
		Exclude: opts.Exclude,
		Classify: func(rel string) string {
			for _, e := range opts.Ecosystems {
				if e.IsManifest(rel) {
					return e.Name
				}
			}
			return ""
		},
	})
	if err != nil {
		return nil, err
	}

	type decl struct {
		eco  *ecosystem.Ecosystem
		dep  manifest.Dependency
		path string
		rel  string
	}
	var decls []decl
	autoScopes := map[string]struct{}{}

	for _, m := range manifests {
		eco := byName[m.Ecosystem]
		if eco == nil {
			continue
		}
		data, err := manifest.ReadFile(m.Path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", m.Path, err)
		}
		deps, err := eco.Parse(m.Rel, data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", m.Path, err)
		}
		for _, d := range deps {
			decls = append(decls, decl{eco: eco, dep: d, path: m.Path, rel: m.Rel})
			if eco.Name == "npm" {
				if s := npm.Scope(d.Name); s != "" {
					autoScopes[s] = struct{}{}
				}
			}
		}
	}

	res := &Result{Stats: Stats{Manifests: len(manifests), Packages: len(decls)}}
	seenFinding := map[string]struct{}{}

	add := func(f Finding) {
		key := f.Ecosystem + "|" + f.Package + "|" + f.Kind + "|" + strings.Join(f.Suggestions, ",")
		if _, ok := seenFinding[key]; ok {
			return
		}
		seenFinding[key] = struct{}{}
		res.Findings = append(res.Findings, f)
	}

	for _, d := range decls {
		name := d.dep.Name
		key := d.eco.Key(name)
		if opts.SafeIgnore != nil && opts.SafeIgnore.Match(name) {
			res.Stats.Skipped++
			continue
		}
		if _, ok := opts.Allow[name]; ok {
			res.Stats.Skipped++
			continue
		}
		if _, ok := opts.Allow[key]; ok {
			res.Stats.Skipped++
			continue
		}
		set := d.eco.Corpus()
		if set.Contains(key) || set.Contains(name) {
			continue
		}
		suggestions := nearest(set, name, key, opts.MaxDistance)
		if len(suggestions) == 0 {
			continue
		}
		dist := distance.Distance(name, suggestions[0])
		if dist == 0 {
			dist = distance.Distance(key, suggestions[0])
		}
		add(Finding{
			Ecosystem:   d.eco.Name,
			Package:     name,
			Version:     d.dep.Version,
			Manifest:    d.rel,
			Line:        d.dep.Line,
			Group:       d.dep.Group,
			Distance:    dist,
			Suggestions: suggestions,
			Kind:        "popular",
			Severity:    "critical",
		})
	}

	// Scope peer checks.
	scopes := map[string]struct{}{}
	for _, s := range opts.Scopes {
		s = strings.TrimSpace(s)
		if s != "" {
			scopes[s] = struct{}{}
		}
	}
	if !opts.NoScopePeers {
		for s := range autoScopes {
			scopes[s] = struct{}{}
		}
	}
	if len(scopes) > 0 {
		byScope := map[string][]decl{}
		for _, d := range decls {
			if d.eco.Name != "npm" {
				continue
			}
			s := npm.Scope(d.dep.Name)
			if _, ok := scopes[s]; !ok {
				continue
			}
			byScope[s] = append(byScope[s], d)
		}
		for _, peers := range byScope {
			for i := 0; i < len(peers); i++ {
				for j := i + 1; j < len(peers); j++ {
					a, b := peers[i], peers[j]
					if a.dep.Name == b.dep.Name {
						continue
					}
					aLeaf := scopeLeaf(a.dep.Name)
					bLeaf := scopeLeaf(b.dep.Name)
					dist := distance.Distance(aLeaf, bLeaf)
					if dist < 1 || dist > opts.MaxDistance {
						continue
					}
					// Prefer flagging the longer leaf (likely typo) against the shorter.
					typo, good := a, b
					if len(bLeaf) > len(aLeaf) || (len(aLeaf) == len(bLeaf) && b.dep.Name > a.dep.Name) {
						typo, good = b, a
					}
					add(Finding{
						Ecosystem:   "npm",
						Package:     typo.dep.Name,
						Version:     typo.dep.Version,
						Manifest:    typo.rel,
						Line:        typo.dep.Line,
						Group:       typo.dep.Group,
						Distance:    dist,
						Suggestions: []string{good.dep.Name},
						Kind:        "scope-peer",
						Severity:    "critical",
					})
				}
			}
		}
	}

	sort.Slice(res.Findings, func(i, j int) bool {
		if res.Findings[i].Ecosystem != res.Findings[j].Ecosystem {
			return res.Findings[i].Ecosystem < res.Findings[j].Ecosystem
		}
		return res.Findings[i].Package < res.Findings[j].Package
	})
	res.Stats.Findings = len(res.Findings)
	return res, nil
}

func nearest(set interface {
	Candidates(name string, maxDist int) []string
}, name, key string, maxDist int) []string {
	type hit struct {
		name string
		dist int
	}
	var hits []hit
	seen := map[string]struct{}{}
	consider := func(cand string) {
		if _, ok := seen[cand]; ok {
			return
		}
		d1 := distance.Distance(name, cand)
		d2 := distance.Distance(key, cand)
		d := d1
		if d2 < d {
			d = d2
		}
		if d < 1 || d > maxDist {
			return
		}
		seen[cand] = struct{}{}
		hits = append(hits, hit{name: cand, dist: d})
	}
	for _, c := range set.Candidates(name, maxDist) {
		consider(c)
	}
	if key != name {
		for _, c := range set.Candidates(key, maxDist) {
			consider(c)
		}
	}
	// Also try separator-stripped length buckets via candidates of stripped form.
	stripped := distance.StripSeparators(name)
	if stripped != name {
		for _, c := range set.Candidates(stripped, maxDist+2) {
			consider(c)
		}
	}
	if len(hits) == 0 {
		return nil
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].dist != hits[j].dist {
			return hits[i].dist < hits[j].dist
		}
		return hits[i].name < hits[j].name
	})
	best := hits[0].dist
	var out []string
	for _, h := range hits {
		if h.dist != best {
			break
		}
		out = append(out, h.name)
		if len(out) >= 3 {
			break
		}
	}
	return out
}

func scopeLeaf(name string) string {
	if i := strings.LastIndex(name, "/"); i >= 0 && i+1 < len(name) {
		return name[i+1:]
	}
	return name
}
