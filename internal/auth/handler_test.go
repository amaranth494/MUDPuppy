package auth

import (
	"bytes"
	"log"
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestOTPNotLogged proves DR-2-01 is closed: the one-time sign-in code
// never reaches the log unless AUTH_LOG_OTP is explicitly turned on, the
// audit fact that a code was issued survives either way, and the
// suppressed line carries no other identifier (email address or code
// fragment) that could reintroduce the same disclosure under a different
// name.
//
// The test values below (e.g. "482913") are placeholders for test
// assertions only, never a real one-time sign-in code.
func TestOTPNotLogged(t *testing.T) {
	t.Run("logs_issuance_without_the_code", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(os.Stderr)
		origFlags := log.Flags()
		log.SetFlags(0) // no timestamp prefix, so digit-run assertions test only the message
		defer log.SetFlags(origFlags)

		h := &Handler{logOTPCode: false}
		h.logOTPIssued("482913")

		out := buf.String()
		if out == "" {
			t.Fatal("expected a log line to be written, got none")
		}
		if !strings.Contains(out, "one-time sign-in code issued") {
			t.Errorf("expected the issuance audit line, got: %q", out)
		}
		if strings.Contains(out, "482913") {
			t.Errorf("the code must not appear in the log when logOTPCode is false, got: %q", out)
		}
	})

	t.Run("logs_the_code_only_when_flag_is_on", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(os.Stderr)
		origFlags := log.Flags()
		log.SetFlags(0)
		defer log.SetFlags(origFlags)

		h := &Handler{logOTPCode: true}
		h.logOTPIssued("482913")

		out := buf.String()
		if !strings.Contains(out, "482913") {
			t.Errorf("expected the code to be printed when logOTPCode is true, got: %q", out)
		}
	})

	t.Run("suppressed_line_carries_no_identifiers", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(os.Stderr)
		origFlags := log.Flags()
		log.SetFlags(0)
		defer log.SetFlags(origFlags)

		h := &Handler{logOTPCode: false}
		h.logOTPIssued("482913")

		out := buf.String()
		if strings.Contains(out, "@") {
			t.Errorf("the suppressed line must not contain an email address, got: %q", out)
		}
		if digitRun := regexp.MustCompile(`[0-9]{4,}`); digitRun.MatchString(out) {
			t.Errorf("the suppressed line must not contain a run of four or more digits (a code fragment), got: %q", out)
		}
	})
}
