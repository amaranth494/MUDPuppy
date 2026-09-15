package policy

import (
	"strings"
	"testing"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "version on third line",
			text: "# Some Policy\nDate: 2026-09-14\nPolicy version: 1.0\n",
			want: "1.0",
		},
		{
			name: "extra surrounding whitespace still trims",
			text: "# Some Policy\n   Policy version:   1.0   \nDate: 2026-09-14\n",
			want: "1.0",
		},
		{
			name: "no header present yields empty string",
			text: "# Some Policy\nDate: 2026-09-14\nNo version here.\n",
			want: "",
		},
		{
			name: "header appearing later in the document is still found",
			text: "# Some Policy\nDate: 2026-09-14\n\n## Intro\nSome text.\n\nPolicy version: 2.3\n",
			want: "2.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseVersion(tt.text)
			if got != tt.want {
				t.Errorf("parseVersion() version = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestVersionOfEmbeddedPolicy(t *testing.T) {
	want := "1.0"
	got := Version()
	if got != want {
		t.Errorf("Version() = %q, want %q", got, want)
	}
}

func TestTextOfEmbeddedPolicy(t *testing.T) {
	text := Text()
	if text == "" {
		t.Fatal("Text() returned empty string")
	}

	wantHeading := "# MUDPuppy AI Player: Safety and Abuse Policy"
	if !strings.Contains(text, wantHeading) {
		t.Errorf("Text() missing heading %q (first line: %q)", wantHeading, firstLine(text))
	}

	wantSection := "## 1. Respect the game's rules"
	if !strings.Contains(text, wantSection) {
		t.Errorf("Text() missing section %q (first line: %q)", wantSection, firstLine(text))
	}
}

// firstLine returns the first line of s, used in failure messages so the
// full policy body is never printed to test output (T-1-11).
func firstLine(s string) string {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return s[:idx]
	}
	return s
}
