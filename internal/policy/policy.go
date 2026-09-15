// Package policy embeds the approved Safety and Abuse policy document and
// exposes its text and version, parsed once at package init from the
// document's own "Policy version:" header. This package holds no
// acceptance-lifetime logic (D-06) and serves no HTTP surface; that
// belongs to a later plan.
package policy

import (
	_ "embed"
	"strings"
)

//go:embed safety-and-abuse-policy-v1.md
var policyText string

var policyVersion = parseVersion(policyText)

// parseVersion scans text for a line beginning with "Policy version:" and
// returns the remainder trimmed of whitespace. It returns the empty string
// if no such header line is present.
func parseVersion(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if v, found := strings.CutPrefix(strings.TrimSpace(line), "Policy version:"); found {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// Text returns the full embedded policy document.
func Text() string {
	return policyText
}

// Version returns the policy version parsed from the embedded document's
// "Policy version:" header.
func Version() string {
	return policyVersion
}
