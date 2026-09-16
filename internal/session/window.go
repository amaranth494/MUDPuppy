package session

import "regexp"

// RecentOutputWindowBytes bounds the per-user ring of recent game text kept
// for the AI driver's snapshot. Roughly two to four 80-column screens of
// plain text. This is a server constant, deliberately not a profile
// setting (D-04) — per-profile tuning of the window size is deferred to
// Phase 4 (03-CONTEXT.md "Deferred Ideas").
const RecentOutputWindowBytes = 8192

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

// ringBuffer is a fixed-size byte ring: the last RecentOutputWindowBytes
// bytes written, oldest-overwritten-first. It holds no mutex and no
// per-user map — callers (Manager) hold m.mu around every method call.
type ringBuffer struct {
	buf    []byte
	offset int  // next write position in buf
	filled bool // true once buf has wrapped at least once
}

// newRingBuffer allocates a ring sized to RecentOutputWindowBytes.
func newRingBuffer() *ringBuffer {
	return &ringBuffer{buf: make([]byte, RecentOutputWindowBytes)}
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
		return
	}
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

// snapshot returns the ring's contents in chronological order (oldest
// first, newest last), or "" when nothing has been written. No locking;
// the caller holds m.mu.
func (r *ringBuffer) snapshot() string {
	if !r.filled {
		return string(r.buf[:r.offset])
	}
	out := make([]byte, len(r.buf))
	n := copy(out, r.buf[r.offset:])
	copy(out[n:], r.buf[:r.offset])
	return string(out)
}
