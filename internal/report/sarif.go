package report

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/url"
	"strings"

	"github.com/omni-line/typosquat-detector/internal/scan"
)

// SARIF 2.1.0 output for GitHub code scanning and other SAST dashboards.
// Spec: https://docs.oasis-open.org/sarif/sarif/v2.1.0/sarif-v2.1.0.html

const (
	sarifSchema  = "https://json.schemastore.org/sarif-2.1.0.json"
	toolName     = "typosquat-detector"
	toolInfoURI  = "https://github.com/omni-line/typosquat-detector"
	fingerprintK = "typosquat/v1"
)

type sarifRule struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	ShortDescription sarifText      `json:"shortDescription"`
	FullDescription  sarifText      `json:"fullDescription"`
	Help             sarifHelp      `json:"help"`
	DefaultConfig    sarifConfig    `json:"defaultConfiguration"`
	Properties       map[string]any `json:"properties"`
	kind             scan.Kind
	severity         scan.Severity
}

type sarifText struct {
	Text string `json:"text"`
}

type sarifHelp struct {
	Text     string `json:"text"`
	Markdown string `json:"markdown"`
}

type sarifConfig struct {
	Level string `json:"level"`
}

// sarifRules has one rule per (kind, severity) so that GitHub, which reads
// security-severity from the rule rather than the result, ranks alerts
// correctly.
var sarifRules = []sarifRule{
	newRule("TSQ001", "PopularPackageOneEdit", scan.KindPopular, scan.SeverityCritical, "9.3",
		"Dependency name is one edit away from a popular package",
		"The dependency is not in the popular-package corpus but is one edit (insert, delete, substitute, swap or separator change) away from a high-download package. This is the most common typosquatting pattern."),
	newRule("TSQ002", "PopularPackageTwoEdits", scan.KindPopular, scan.SeverityHigh, "7.5",
		"Dependency name is two edits away from a popular package",
		"The dependency is not in the popular-package corpus but is two edits away from a high-download package. Verify that the name is intended."),
	newRule("TSQ003", "NamespacePeerOneEdit", scan.KindScopePeer, scan.SeverityHigh, "7.0",
		"Dependency name is one edit away from another package in the same namespace",
		"Two packages in the same namespace (for example an npm @scope) are one edit apart. One of them may be a typo that resolves to an attacker-controlled package."),
	newRule("TSQ004", "NamespacePeerTwoEdits", scan.KindScopePeer, scan.SeverityMedium, "5.0",
		"Dependency name is two edits away from another package in the same namespace",
		"Two packages in the same namespace are two edits apart. Verify that both names are intended."),
}

func newRule(id, name string, kind scan.Kind, sev scan.Severity, score, short, full string) sarifRule {
	help := full + "\n\nIf the name is intended, suppress it with --allow <name>; otherwise replace it with the package you meant."
	return sarifRule{
		ID:               id,
		Name:             name,
		ShortDescription: sarifText{Text: short},
		FullDescription:  sarifText{Text: full},
		Help:             sarifHelp{Text: help, Markdown: help},
		DefaultConfig:    sarifConfig{Level: sarifLevel(sev)},
		Properties: map[string]any{
			"security-severity": score,
			"tags":              []string{"security", "supply-chain", "typosquatting"},
			"precision":         "medium",
		},
		kind:     kind,
		severity: sev,
	}
}

func ruleFor(f scan.Finding) (int, sarifRule) {
	for i, r := range sarifRules {
		if r.kind == f.Kind && r.severity == f.Severity {
			return i, r
		}
	}
	return 0, sarifRules[0]
}

func sarifLevel(s scan.Severity) string {
	if s.AtLeast(scan.SeverityHigh) {
		return "error"
	}
	return "warning"
}

type sarifLog struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool        sarifTool         `json:"tool"`
	Invocations []sarifInvocation `json:"invocations"`
	Results     []sarifResult     `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version,omitempty"`
	InformationURI string      `json:"informationUri"`
	Rules          []sarifRule `json:"rules"`
}

type sarifInvocation struct {
	ExecutionSuccessful bool                `json:"executionSuccessful"`
	Notifications       []sarifNotification `json:"toolExecutionNotifications,omitempty"`
}

type sarifNotification struct {
	Level     string          `json:"level"`
	Message   sarifText       `json:"message"`
	Locations []sarifLocation `json:"locations,omitempty"`
}

type sarifResult struct {
	RuleID              string            `json:"ruleId"`
	RuleIndex           int               `json:"ruleIndex"`
	Level               string            `json:"level"`
	Message             sarifText         `json:"message"`
	Locations           []sarifLocation   `json:"locations"`
	PartialFingerprints map[string]string `json:"partialFingerprints"`
	Properties          map[string]any    `json:"properties"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysical `json:"physicalLocation"`
}

type sarifPhysical struct {
	ArtifactLocation sarifArtifact `json:"artifactLocation"`
	Region           *sarifRegion  `json:"region,omitempty"`
}

type sarifArtifact struct {
	URI       string `json:"uri"`
	URIBaseID string `json:"uriBaseId"`
}

type sarifRegion struct {
	StartLine int `json:"startLine"`
}

func sarifLoc(rel string, line int) sarifLocation {
	loc := sarifLocation{PhysicalLocation: sarifPhysical{
		ArtifactLocation: sarifArtifact{URI: (&url.URL{Path: rel}).EscapedPath(), URIBaseID: "%SRCROOT%"},
	}}
	if line > 0 {
		loc.PhysicalLocation.Region = &sarifRegion{StartLine: line}
	}
	return loc
}

func writeSARIF(w io.Writer, res *scan.Result, opts Options) error {
	results := make([]sarifResult, 0, len(res.Findings))
	for _, f := range res.Findings {
		idx, rule := ruleFor(f)
		sum := sha256.Sum256([]byte(strings.Join([]string{f.Ecosystem, f.Manifest, f.Package, string(f.Kind)}, "\x00")))
		props := map[string]any{
			"ecosystem":   f.Ecosystem,
			"package":     f.Package,
			"suggestions": f.Suggestions,
			"distance":    f.Distance,
			"severity":    f.Severity,
			"kind":        f.Kind,
		}
		for k, v := range map[string]string{
			"version": f.Version, "group": f.Group, "technique": string(f.Technique),
			"purl": f.PURL, "registry_url": f.RegistryURL, "suggestion_url": f.SuggestionURL,
		} {
			if v != "" {
				props[k] = v
			}
		}
		results = append(results, sarifResult{
			RuleID:              rule.ID,
			RuleIndex:           idx,
			Level:               sarifLevel(f.Severity),
			Message:             sarifText{Text: f.Message},
			Locations:           []sarifLocation{sarifLoc(f.Manifest, f.Line)},
			PartialFingerprints: map[string]string{fingerprintK: hex.EncodeToString(sum[:])},
			Properties:          props,
		})
	}
	inv := sarifInvocation{ExecutionSuccessful: true}
	for _, wn := range res.Warnings {
		inv.Notifications = append(inv.Notifications, sarifNotification{
			Level:     "warning",
			Message:   sarifText{Text: wn.Message},
			Locations: []sarifLocation{sarifLoc(wn.Manifest, 0)},
		})
	}
	doc := sarifLog{
		Schema:  sarifSchema,
		Version: "2.1.0",
		Runs: []sarifRun{{
			Tool: sarifTool{Driver: sarifDriver{
				Name:           toolName,
				Version:        opts.Version,
				InformationURI: toolInfoURI,
				Rules:          sarifRules,
			}},
			Invocations: []sarifInvocation{inv},
			Results:     results,
		}},
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(doc)
}
