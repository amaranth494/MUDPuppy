package session

import (
	"regexp"
	"time"
)

// RecentOutputWindowBytes bounds the per-user ring of recent game text kept
// for the AI driver's snapshot. Roughly two to four 80-column screens of
// plain text. This is a server constant, deliberately not a profile
// setting (D-04) — per-profile tuning of the window size is deferred to
// Phase 4 (03-CONTEXT.md "Deferred Ideas").
const RecentOutputWindowBytes = 8192

// windowMaxAge is the Immediate Context layer's age bound (D-09,
// .specify/specs/ai-memory-model-v1.md §1): roughly how long ago game text
// may have arrived and still count as "what just happened" for a decision.
// It is what makes the window a reaction window rather than a scroll-back.
// Approximate by intent — chunk-boundary granularity, not per-byte.
const windowMaxAge = 10 * time.Second

// windowMinRetainedBytes is the Immediate Context layer's never-empty
// floor (D-09): the smallest amount of the most recent screenful kept
// regardless of age, so an engage after a long quiet spell still sees the
// room the character is standing in rather than nothing. Approximate by
// intent, same as windowMaxAge.
const windowMinRetainedBytes = 2048

// ansiCSI matches an ANSI CSI (Control Sequence Introducer) escape
// sequence: ESC '[' followed by zero or more parameter bytes and one
// final letter. Sibling transform to the existing telnet IAC stripper in
// websocket.go, same pure []byte -> []byte shape.
var ansiCSI = regexp.MustCompile("\x1b\\[[0-9;]*[a-zA-Z]")

// stripANSI removes ANSI CSI colour/formatting sequences from b, returning
// a new slice. A CSI sequence split across two socket reads (the escape
// byte in one read, the rest in the next) will not match here and leaves a
// few stray bytes in the window — this is cosmetic noise for a language
// model to ignore, not a correctness bug, and is deliberately not worth a
// terminal-emulation state machine (RESEARCH Pattern 1). This function is
// for the AI's snapshot only; it is never applied to the browser-bound
// stream, which must keep receiving full ANSI so xterm.js renders colour.
func stripANSI(b []byte) []byte {
	return ansiCSI.ReplaceAll(b, nil)
}

// chunkMarker records where one append() call's bytes ended (as a
// cumulative byte count since the ring was created) and when it happened,
// so snapshotRecent can tell how much of the ring's current contents
// arrived within an age bound. Granularity is per-append-call, not
// per-byte — approximate by design (D-09).
type chunkMarker struct {
	cumEnd int64 // r.totalWritten immediately after this append
	at     time.Time
}

// ringBuffer is a fixed-size byte ring: the last RecentOutputWindowBytes
// bytes written, oldest-overwritten-first. It holds no mutex and no
// per-user map — callers (Manager) hold m.mu around every method call.
type ringBuffer struct {
	buf    []byte
	offset int  // next write position in buf
	filled bool // true once buf has wrapped at least once

	// now returns the current time, defaulting to time.Now. A test-only
	// seam: swapped for an injected clock so age-bound tests drive time
	// directly instead of sleeping (no fake-clock library is added).
	now func() time.Time

	// totalWritten is the cumulative count of bytes ever appended, used
	// only to place chunkMarker.cumEnd on a monotonic scale; it is not a
	// size bound (RecentOutputWindowBytes remains the only byte ceiling).
	totalWritten int64

	// chunks records one marker per append call, oldest first, trimmed as
	// soon as every byte it refers to has been overwritten in buf.
	chunks []chunkMarker
}

// newRingBuffer allocates a ring sized to RecentOutputWindowBytes.
func newRingBuffer() *ringBuffer {
	return &ringBuffer{buf: make([]byte, RecentOutputWindowBytes), now: time.Now}
}

// append copies p into the ring, oldest-first-overwritten. When len(p) is
// larger than the ring, only the trailing RecentOutputWindowBytes bytes of
// p are kept. No locking; the caller holds m.mu.
func (r *ringBuffer) append(p []byte) {
	if len(p) == 0 {
		return
	}
	if len(p) >= len(r.buf) {
		copy(r.buf, p[len(p)-len(r.buf):])
		r.offset = 0
		r.filled = true
	} else {
		n := copy(r.buf[r.offset:], p)
		if n < len(p) {
			// Wrapped: the rest goes at the start of the buffer.
			copy(r.buf, p[n:])
			r.filled = true
		}
		r.offset = (r.offset + len(p)) % len(r.buf)
		if r.offset == 0 {
			r.filled = true
		}
	}
	r.recordChunk(len(p))
}

// recordChunk appends a chunk marker for an append of n bytes that just
// landed, then trims markers from the front whose bytes have since been
// entirely overwritten (their cumEnd is more than a full ring capacity
// behind the current write position) — the parallel-slice half of D-09's
// time dimension, kept in step with the byte ring's own overwrite rule.
func (r *ringBuffer) recordChunk(n int) {
	r.totalWritten += int64(n)
	r.chunks = append(r.chunks, chunkMarker{cumEnd: r.totalWritten, at: r.now()})

	cap64 := int64(len(r.buf))
	for len(r.chunks) > 0 && r.totalWritten-r.chunks[0].cumEnd >= cap64 {
		r.chunks = r.chunks[1:]
	}
}

// snapshotBytes returns the ring's contents in chronological order (oldest
// first, newest last) as a fresh []byte, or an empty slice when nothing
// has been written. No locking; the caller holds m.mu. snapshot and
// snapshotRecent both build on this one chronological-ordering routine.
func (r *ringBuffer) snapshotBytes() []byte {
	if !r.filled {
		out := make([]byte, r.offset)
		copy(out, r.buf[:r.offset])
		return out
	}
	out := make([]byte, len(r.buf))
	n := copy(out, r.buf[r.offset:])
	copy(out[n:], r.buf[:r.offset])
	return out
}

// snapshot returns the ring's contents in chronological order (oldest
// first, newest last), or "" when nothing has been written. No locking;
// the caller holds m.mu.
func (r *ringBuffer) snapshot() string {
	return string(r.snapshotBytes())
}

// snapshotRecent returns the ring's contents in chronological order,
// trimmed to whatever arrived within maxAge of the ring's current time —
// except that when that is fewer than minBytes, it returns the most
// recent minBytes worth instead, so it is never empty while the ring
// holds anything at all (D-09's never-empty floor). It never returns more
// than the ring's own byte ceiling (RecentOutputWindowBytes), since it can
// only ever return bytes already present in the ring. No locking; the
// caller holds m.mu.
func (r *ringBuffer) snapshotRecent(maxAge time.Duration, minBytes int) string {
	full := r.snapshotBytes()
	if len(full) == 0 {
		return ""
	}

	cutoff := r.now().Add(-maxAge)
	startOfFull := r.totalWritten - int64(len(full))

	// Walk chunk markers newest-to-oldest. boundary tracks the cumulative
	// byte position just before the oldest chunk seen so far that still
	// arrived within the age bound; it stops updating the moment an older
	// chunk is found, since chunks are time-ordered.
	boundary := r.totalWritten
	for i := len(r.chunks) - 1; i >= 0; i-- {
		if r.chunks[i].at.Before(cutoff) {
			break
		}
		if i == 0 {
			boundary = startOfFull
		} else {
			boundary = r.chunks[i-1].cumEnd
		}
	}
	if boundary < startOfFull {
		boundary = startOfFull
	}

	recentBytes := r.totalWritten - boundary
	if recentBytes < int64(minBytes) {
		recentBytes = int64(minBytes)
	}
	if recentBytes > int64(len(full)) {
		recentBytes = int64(len(full))
	}

	return string(full[int64(len(full))-recentBytes:])
}
