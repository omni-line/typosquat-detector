package scan

import (
	"fmt"
	"strings"
)

// Kind identifies which check produced a finding.
type Kind string

// Finding kinds.
const (
	// KindPopular: the name is a near-miss of a high-download package.
	KindPopular Kind = "popular"
	// KindScopePeer: the name is a near-miss of another package in the same
	// namespace (e.g. npm scope) declared in the project.
	KindScopePeer Kind = "scope-peer"
)

// Severity grades how likely a finding is to be a real attack.
type Severity string

// Severities, from most to least severe.
const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
)

// Severities lists all severities from most to least severe.
var Severities = []Severity{SeverityCritical, SeverityHigh, SeverityMedium}

// Rank orders severities; higher is more severe. Unknown values rank 0.
func (s Severity) Rank() int {
	switch s {
	case SeverityCritical:
		return 3
	case SeverityHigh:
		return 2
	case SeverityMedium:
		return 1
	default:
		return 0
	}
}

// AtLeast reports whether s is as severe as min.
func (s Severity) AtLeast(min Severity) bool {
	return s.Rank() >= min.Rank()
}

// ParseSeverity parses a severity name case-insensitively.
func ParseSeverity(v string) (Severity, error) {
	s := Severity(strings.ToLower(strings.TrimSpace(v)))
	if s.Rank() == 0 {
		return "", fmt.Errorf("unknown severity %q (want critical|high|medium)", v)
	}
	return s, nil
}

// severityFor grades a finding. One edit from a popular package is the
// classic attack (crossenv, reqeusts); two edits is more often a coincidence.
// Namespace peers are usually internal packages, so they rank one step lower.
func severityFor(kind Kind, dist int) Severity {
	switch {
	case kind == KindPopular && dist <= 1:
		return SeverityCritical
	case kind == KindPopular, dist <= 1:
		return SeverityHigh
	default:
		return SeverityMedium
	}
}
