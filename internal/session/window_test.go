package session

import (
	"fmt"
	"strings"
	"testing"
)

// TestWindowSnapshot exercises stripANSI and ringBuffer.snapshot in the
// table-driven t.Run style of internal/icm/icm_test.go.
func TestWindowSnapshot(t *testing.T) {
	t.Run("strips_ansi_colour_codes", func(t *testing.T) {
		r := newRingBuffer()
		r.append(stripANSI([]byte("\x1b[32mgreen\x1b[0m plain \x1b[1;33mwords\x1b[0m")))

		got := r.snapshot()
		if strings.Contains(got, "\x1b") {
			t.Fatalf("snapshot contains an escape byte: %q", got)
		}
		if !strings.Contains(got, "green") || !strings.Contains(got, "plain") || !strings.Contains(got, "words") {
			t.Fatalf("snapshot missing expected words: %q", got)
		}
	})

	t.Run("keeps_plain_text_untouched", func(t *testing.T) {
		r := newRingBuffer()
		line := "You are standing in an open field.\n"
		r.append(stripANSI([]byte(line)))

		if got := r.snapshot(); got != line {
			t.Fatalf("snapshot = %q, want %q", got, line)
		}
	})

	t.Run("split_escape_sequence_is_tolerated", func(t *testing.T) {
		r := newRingBuffer()
		// A CSI sequence split across two socket reads will not match the
		// regex on either half — the stray fragment "\x1b[3" is accepted
		// into the window as cosmetic noise, not a crash (RESEARCH Pattern
		// 1). What matters is the call does not panic and the plain text
		// that follows is still present.
		defer func() {
			if rec := recover(); rec != nil {
				t.Fatalf("append/stripANSI panicked on a split escape sequence: %v", rec)
			}
		}()
		r.append(stripANSI([]byte("\x1b[3")))
		r.append(stripANSI([]byte("2mhello")))

		got := r.snapshot()
		if !strings.Contains(got, "hello") {
			t.Fatalf("snapshot missing %q: got %q", "hello", got)
		}
	})

	t.Run("empty_ring_snapshots_to_empty_string", func(t *testing.T) {
		r := newRingBuffer()
		if got := r.snapshot(); got != "" {
			t.Fatalf("snapshot = %q, want empty string", got)
		}
	})
}

// TestWindowRingBound proves the ring is bounded: after more than
// RecentOutputWindowBytes is written, the oldest bytes are gone and the
// newest bytes are present, and the snapshot length is exactly
// RecentOutputWindowBytes (an exact equality, not a bound check).
func TestWindowRingBound(t *testing.T) {
	r := newRingBuffer()

	total := RecentOutputWindowBytes + 500
	written := make([]byte, 0, total)

	// Write distinguishable bytes across several calls so the wrap-around
	// bookkeeping (not just a single big append) is exercised. Each chunk
	// carries a unique numbered label so a plain byte-value cycle (mod 256)
	// can't accidentally reappear later and produce a false "still
	// present" match.
	chunkSize := 137 // an odd size so chunks don't align with the ring bound
	chunkIdx := 0
	for len(written) < total {
		label := []byte(fmt.Sprintf("|chunk%05d|", chunkIdx))
		chunk := make([]byte, chunkSize)
		copy(chunk, label)
		for i := len(label); i < chunkSize; i++ {
			chunk[i] = '.'
		}

		n := len(chunk)
		if len(written)+n > total {
			n = total - len(written)
		}
		r.append(chunk[:n])
		written = append(written, chunk[:n]...)
		chunkIdx++
	}

	got := r.snapshot()
	if len(got) != RecentOutputWindowBytes {
		t.Fatalf("snapshot length = %d, want exactly %d", len(got), RecentOutputWindowBytes)
	}

	// The newest bytes written must be present, at the end of the snapshot.
	wantTail := written[len(written)-RecentOutputWindowBytes:]
	if got != string(wantTail) {
		t.Fatalf("snapshot does not equal the trailing %d bytes written", RecentOutputWindowBytes)
	}

	// The very first chunk's unique label — the oldest bytes written — must
	// be entirely gone from the snapshot.
	firstLabel := "|chunk00000|"
	if strings.Contains(got, firstLabel) {
		t.Fatalf("snapshot still contains the first chunk's label %q; oldest bytes were not overwritten", firstLabel)
	}

	// The very last chunk's unique label — the newest bytes written — must
	// be present.
	lastLabel := fmt.Sprintf("|chunk%05d|", chunkIdx-1)
	if !strings.Contains(got, lastLabel) {
		t.Fatalf("snapshot missing the last chunk's label %q; newest bytes were lost", lastLabel)
	}
}

// TestWindowIncludesPreEngageText proves D-04's "including text from
// before the switch was flipped" needs no special-casing: the window is
// fed continuously by appendOutputWindow regardless of autopilot state, so
// game text that arrived before #AUTO ON is already in the snapshot by the
// time the driver takes its first look.
func TestWindowIncludesPreEngageText(t *testing.T) {
	m := newTestManager()
	const userID = "pre-engage-user"
	seedConnectedSession(m, userID, "conn-1")

	// Push several chunks through appendOutputWindow, standing in for game
	// output arriving before autopilot was ever switched on.
	chunks := []string{
		"You wake up in a dim room.\n",
		"A rat scurries past.\n",
		"Exits: north, east.\n",
	}
	for _, c := range chunks {
		m.appendOutputWindow(userID, []byte(c))
	}

	got := m.RecentOutputSnapshot(userID)
	lastEnd := -1
	for _, c := range chunks {
		idx := strings.Index(got, c)
		if idx == -1 {
			t.Fatalf("snapshot missing pre-engage chunk %q: got %q", c, got)
		}
		if idx <= lastEnd {
			t.Fatalf("chunk %q is out of order in snapshot %q", c, got)
		}
		lastEnd = idx
	}
}

// TestWindowSurvivesDisconnect drives the real Manager.Disconnect, never a
// bare Session struct — a test that asserted on a constructed struct would
// pass while the production path silently lost the window (RESEARCH
// Pitfall 4). Proves D-02: a WAITING-to-ON resume must read text that
// spans the drop.
func TestWindowSurvivesDisconnect(t *testing.T) {
	m := newTestManager()
	const userID = "survives-disconnect-user"
	seedConnectedSession(m, userID, "conn-1")

	m.appendOutputWindow(userID, []byte("The room is quiet before the drop.\n"))

	if err := m.Disconnect(userID, ReasonRemote); err != nil {
		t.Fatalf("Disconnect err = %v, want nil", err)
	}

	got := m.RecentOutputSnapshot(userID)
	if !strings.Contains(got, "The room is quiet before the drop.") {
		t.Fatalf("RecentOutputSnapshot after Disconnect = %q, want it to still contain the pre-drop text", got)
	}
}
