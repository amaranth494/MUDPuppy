package session

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// windowTestClock is an injectable clock a test can drive directly,
// standing in for windowMaxAge/windowMinRetainedBytes's ringBuffer.now
// field so age-bound tests never sleep (04-RESEARCH Don't Hand-Roll: no
// fake-clock library is added).
type windowTestClock struct {
	t time.Time
}

func (c *windowTestClock) now() time.Time { return c.t }

func (c *windowTestClock) advance(d time.Duration) { c.t = c.t.Add(d) }

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

// TestWindow_AgeBounded proves D-09's age bound: snapshotRecent returns
// text that arrived within windowMaxAge of the ring's current time and
// excludes text from a minute earlier, even though the ring itself still
// holds both (proven separately via the plain snapshot()). chunk-new is
// padded past windowMinRetainedBytes so the never-empty floor (proven
// separately by TestWindow_NeverEmptyAfterQuiet) cannot mask the age
// exclusion by pulling chunk-old back in to meet the floor.
func TestWindow_AgeBounded(t *testing.T) {
	r := newRingBuffer()
	clock := &windowTestClock{t: time.Now()}
	r.now = clock.now

	r.append([]byte("chunk-old: a minute ago\n"))
	clock.advance(40 * time.Second)
	r.append([]byte("chunk-mid: forty seconds later\n"))
	clock.advance(15 * time.Second)
	padding := strings.Repeat("x", windowMinRetainedBytes)
	r.append([]byte("chunk-new: fifteen seconds after that\n" + padding))
	// Read a fourth time, a few seconds after the newest chunk landed —
	// well inside windowMaxAge of chunk-new, well outside it for chunk-old.
	clock.advance(3 * time.Second)

	got := r.snapshotRecent(windowMaxAge, windowMinRetainedBytes)
	if !strings.Contains(got, "chunk-new") {
		t.Fatalf("snapshotRecent missing the within-age chunk: got %q", got)
	}
	if strings.Contains(got, "chunk-old") {
		t.Fatalf("snapshotRecent still contains a minute-old chunk: got %q", got)
	}

	// The ring itself — the plain, non-age-aware snapshot — still holds
	// both the old and the new chunk; only the age-aware read trims them.
	full := r.snapshot()
	if !strings.Contains(full, "chunk-old") || !strings.Contains(full, "chunk-new") {
		t.Fatalf("ring's own snapshot() should still hold both chunks: got %q", full)
	}
}

// TestWindow_NeverEmptyAfterQuiet proves D-09's never-empty floor: after a
// quiet spell far longer than windowMaxAge, snapshotRecent still returns
// the most recent bytes rather than an empty string.
func TestWindow_NeverEmptyAfterQuiet(t *testing.T) {
	r := newRingBuffer()
	clock := &windowTestClock{t: time.Now()}
	r.now = clock.now

	r.append([]byte("You are standing in the town square.\n"))
	clock.advance(5 * time.Minute) // far past windowMaxAge; nothing else arrives

	got := r.snapshotRecent(windowMaxAge, windowMinRetainedBytes)
	if got == "" {
		t.Fatalf("snapshotRecent = empty string after a quiet spell, want the most recent screenful")
	}
	if !strings.Contains(got, "town square") {
		t.Fatalf("snapshotRecent = %q, want it to still contain the pre-quiet text", got)
	}
}

// TestWindow_CeilingStillHolds proves T-4-24: the byte ceiling
// (RecentOutputWindowBytes) still wins over the age bound when a flood of
// game text all arrives within windowMaxAge.
func TestWindow_CeilingStillHolds(t *testing.T) {
	r := newRingBuffer()
	clock := &windowTestClock{t: time.Now()}
	r.now = clock.now

	total := RecentOutputWindowBytes + 4096
	written := 0
	chunkSize := 256
	for written < total {
		n := chunkSize
		if written+n > total {
			n = total - written
		}
		chunk := make([]byte, n)
		for i := range chunk {
			chunk[i] = byte('a' + (written+i)%26)
		}
		r.append(chunk)
		written += n
		// All chunks land well within windowMaxAge of each other and of
		// the eventual read.
		clock.advance(100 * time.Millisecond)
	}

	got := r.snapshotRecent(windowMaxAge, windowMinRetainedBytes)
	if len(got) > RecentOutputWindowBytes {
		t.Fatalf("snapshotRecent length = %d, want at most %d (the byte ceiling must still win when both bounds apply)", len(got), RecentOutputWindowBytes)
	}
}

// TestWindow_RecentSnapshotStripsANSI proves the age-aware read strips
// ANSI exactly as the existing snapshot does — both build on the same
// snapshotBytes routine over a ring that already holds stripped bytes.
func TestWindow_RecentSnapshotStripsANSI(t *testing.T) {
	r := newRingBuffer()
	r.append(stripANSI([]byte("\x1b[32mgreen\x1b[0m plain \x1b[1;33mwords\x1b[0m")))

	got := r.snapshotRecent(windowMaxAge, windowMinRetainedBytes)
	if strings.Contains(got, "\x1b") {
		t.Fatalf("snapshotRecent contains an escape byte: %q", got)
	}
	if !strings.Contains(got, "green") || !strings.Contains(got, "plain") || !strings.Contains(got, "words") {
		t.Fatalf("snapshotRecent missing expected words: %q", got)
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
