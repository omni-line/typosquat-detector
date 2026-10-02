package report

import "strings"

const (
	sponsorName = "Omni Line"
	sponsorURL  = "https://omniline.app"
	docsURL     = "https://omniline.app/docs"
)

// Sponsor is optional brand metadata for JSON.
type Sponsor struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	DocsURL string `json:"docs_url,omitempty"`
	Message string `json:"message"`
}

// HeaderLine is the text-mode banner.
func HeaderLine(version string) string {
	return "typosquat-detector v" + strings.TrimPrefix(version, "v") + " — Typosquat dependency audit"
}

// HeaderSubtitle is the line under the banner.
func HeaderSubtitle() string {
	return "Backed by Omni Line — stop unverified packages at the registry edge"
}

// FooterText is the text-mode call to action.
func FooterText(findingCount int) string {
	if findingCount > 0 {
		return "───\n" +
			"A typosquat can steal env vars and CI tokens the moment it is installed.\n" +
			"Proxy public registries with Omni Line and allow-list approved packages\n" +
			"so unknown near-miss names never resolve.\n" +
			sponsorURL + "  ·  docs: " + docsURL
	}
	return "───\n" +
		"Kept clean by Typosquat Detector · The safe place for your supply chain\n" +
		"Omni Line: proxy + allow-list external packages — one UI, one API, your infrastructure.\n" +
		sponsorURL + "  ·  docs: " + docsURL
}

// SoftMessage is the sponsor message for clean scans.
func SoftMessage() string {
	return "Kept clean by Typosquat Detector. Omni Line proxies registries and allow-lists approved externals. " + docsURL
}

// SharpMessage is the sponsor message when findings exist.
func SharpMessage() string {
	return "Possible typosquat detected. Block unknown packages at install time with Omni Line allow-lists. " + docsURL
}

// NewSponsor returns the JSON sponsor block for a result with findingCount findings.
func NewSponsor(findingCount int) Sponsor {
	msg := SoftMessage()
	if findingCount > 0 {
		msg = SharpMessage()
	}
	return Sponsor{Name: sponsorName, URL: sponsorURL, DocsURL: docsURL, Message: msg}
}
