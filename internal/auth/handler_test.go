package auth

import (
	"bytes"
	"errors"
	"log"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/amaranth494/MudPuppy/internal/store"
	"github.com/google/uuid"
)

// fakeUserStore is a minimal UserStore fake for the D-31 login-boundary
// tests below (mirrors internal/driver's fakes): no database connection,
// just enough method surface to satisfy the auth.UserStore interface.
type fakeUserStore struct {
	markLoginStartCalls int
	lastUserID          uuid.UUID
	markLoginStartErr   error
}

func (f *fakeUserStore) Create(email string) (*store.User, error) { return nil, nil }

func (f *fakeUserStore) GetByEmail(email string) (*store.User, error) { return nil, nil }

func (f *fakeUserStore) GetByID(id uuid.UUID) (*store.User, error) { return nil, nil }

func (f *fakeUserStore) MarkLoginStarted(userID uuid.UUID) error {
	f.markLoginStartCalls++
	f.lastUserID = userID
	return f.markLoginStartErr
}

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

// TestMarkLoginStart proves D-31's login-boundary call sites without a
// database or a redis connection: Login calls markLoginStart once at the
// "login" stage, Logout calls it once at the "logout" stage (see the
// wiring in Login and Logout), and a UserStore error at either site is
// swallowed -- logged with ids and stage only, never returned -- so it can
// never break the login or logout response the owner is waiting on.
func TestMarkLoginStart(t *testing.T) {
	t.Run("login_marks_once", func(t *testing.T) {
		fake := &fakeUserStore{}
		h := &Handler{userStore: fake}
		userID := uuid.New()

		h.markLoginStart(userID, "login")

		if fake.markLoginStartCalls != 1 {
			t.Errorf("expected MarkLoginStarted called once, got %d", fake.markLoginStartCalls)
		}
		if fake.lastUserID != userID {
			t.Errorf("expected MarkLoginStarted called with %s, got %s", userID, fake.lastUserID)
		}
	})

	t.Run("logout_marks_once", func(t *testing.T) {
		fake := &fakeUserStore{}
		h := &Handler{userStore: fake}
		userID := uuid.New()

		h.markLoginStart(userID, "logout")

		if fake.markLoginStartCalls != 1 {
			t.Errorf("expected MarkLoginStarted called once, got %d", fake.markLoginStartCalls)
		}
		if fake.lastUserID != userID {
			t.Errorf("expected MarkLoginStarted called with %s, got %s", userID, fake.lastUserID)
		}
	})

	t.Run("store_error_does_not_break_the_caller", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(os.Stderr)
		origFlags := log.Flags()
		log.SetFlags(0)
		defer log.SetFlags(origFlags)

		fake := &fakeUserStore{markLoginStartErr: errors.New("boom: connection refused")}
		h := &Handler{userStore: fake}
		userID := uuid.New()

		// markLoginStart has no return value, so "does not break the
		// caller" means it must not panic even when the store errors, and
		// there is nothing for Login/Logout to check or fail on.
		h.markLoginStart(userID, "login")

		out := buf.String()
		if !strings.Contains(out, "failed to mark login start") {
			t.Errorf("expected a failure log line, got %q", out)
		}
		if !strings.Contains(out, "stage=login") {
			t.Errorf("expected the failure log line to carry the stage, got %q", out)
		}
		if !strings.Contains(out, userID.String()) {
			t.Errorf("expected the failure log line to carry the user id, got %q", out)
		}
		if strings.Contains(out, "boom") {
			t.Errorf("the failure log line must carry ids and stage only, not the underlying error text, got %q", out)
		}
	})

	// Code review WR-08 of Phase 5: AI-chatter's call count is per login, so
	// the driver is told at the same boundary -- sign-in and sign-out alike,
	// and even when the store call failed.
	t.Run("the_login_boundary_hook_is_told_at_sign_in_and_sign_out", func(t *testing.T) {
		var told []string
		h := &Handler{userStore: &fakeUserStore{}}
		h.SetLoginBoundaryHook(func(userID string) { told = append(told, userID) })
		userID := uuid.New()

		h.markLoginStart(userID, "login")
		h.markLoginStart(userID, "logout")

		if len(told) != 2 || told[0] != userID.String() || told[1] != userID.String() {
			t.Fatalf("hook told %v, want the user id twice", told)
		}
	})

	t.Run("the_hook_is_told_even_when_the_store_call_fails", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)
		defer log.SetOutput(os.Stderr)

		told := 0
		h := &Handler{userStore: &fakeUserStore{markLoginStartErr: errors.New("boom")}}
		h.SetLoginBoundaryHook(func(string) { told++ })

		h.markLoginStart(uuid.New(), "login")

		if told != 1 {
			t.Fatalf("hook told %d time(s), want 1", told)
		}
	})

	t.Run("no_hook_is_a_no_op", func(t *testing.T) {
		h := &Handler{userStore: &fakeUserStore{}}
		h.markLoginStart(uuid.New(), "login") // must not panic
	})
}
