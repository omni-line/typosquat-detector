package report

import (
	"encoding/json"
	"io"

	"github.com/omni-line/typosquat-detector/internal/corpus"
	"github.com/omni-line/typosquat-detector/internal/scan"
)

// JSONSchemaVersion is bumped on breaking changes to JSONDocument. Additive
// fields do not bump it.
const JSONSchemaVersion = 1

// JSONDocument is the --format json output.
type JSONDocument struct {
	SchemaVersion int            `json:"schema_version"`
	Version       string         `json:"version"`
	Scan          *ScanInfo      `json:"scan,omitempty"`
	Findings      []scan.Finding `json:"findings"`
	Warnings      []scan.Warning `json:"warnings"`
	Stats         scan.Stats     `json:"stats"`
	Sponsor       *Sponsor       `json:"sponsor,omitempty"`
}

// ScanInfo records how a scan was run, for reproducibility.
type ScanInfo struct {
	Root        string       `json:"root,omitempty"`
	MaxDistance int          `json:"max_distance,omitempty"`
	DurationMS  int64        `json:"duration_ms"`
	Corpus      *corpus.Meta `json:"corpus,omitempty"`
}

func writeJSON(w io.Writer, res *scan.Result, opts Options) error {
	doc := JSONDocument{
		SchemaVersion: JSONSchemaVersion,
		Version:       opts.Version,
		Scan: &ScanInfo{
			Root:        opts.Root,
			MaxDistance: opts.MaxDistance,
			DurationMS:  opts.Duration.Milliseconds(),
			Corpus:      opts.Corpus,
		},
		Findings: res.Findings,
		Warnings: res.Warnings,
		Stats:    res.Stats,
	}
	if doc.Findings == nil {
		doc.Findings = []scan.Finding{}
	}
	if doc.Warnings == nil {
		doc.Warnings = []scan.Warning{}
	}
	if shouldShowMarketing(opts) {
		s := NewSponsor(len(res.Findings))
		doc.Sponsor = &s
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(doc)
}
