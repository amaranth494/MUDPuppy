package session

import (
	"bytes"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// transcriptChannelCapacity bounds the per-session buffered channel that
// decouples the MUD read/write path from the database (T-3-26): once full,
// enqueue drops rather than blocks.
const transcriptChannelCapacity = 512

// transcriptBatchSize and transcriptBatchTick bound how much the writer
// goroutine accumulates before calling AppendGameLines: whichever comes
// first, 100 lines or a 2-second tick.
const (
	transcriptBatchSize = 100
	transcriptBatchTick = 2 * time.Second
)

// TranscriptLine is one line of a session transcript, kept in
// internal/session independent of internal/store's row shape so this
// package never imports internal/store (the interface lives in the
// consuming package, not the producing one). cmd/server/main.go adapts
// this to store.GameSessionLine when wiring the sink.
type TranscriptLine struct {
	Seq       int64
	Source    string // "human", "ai", "game" or "marker" — no other value
	Text      string
	CreatedAt time.Time
}

// TranscriptSink is the store-backed side of the per-connection transcript
// tap (D-14). Implemented by *store.TranscriptStore via a thin adapter
// constructed in cmd/server/main.go and wired in with
// Manager.SetTranscriptSink.
type TranscriptSink interface {
	OpenGameSession(userID, connectionID uuid.UUID) (uuid.UUID, error)
	AppendGameLines(gameSessionID uuid.UUID, lines []TranscriptLine) error
	CloseGameSession(gameSessionID uuid.UUID) error
}

// transcriptSession is the open per-connection transcript tap: a buffered
// channel drained by a single batching writer goroutine, so the MUD read
// loop (ReadOutput) and the send path (SendCommandAs) never block on the
// database (T-3-26). It holds no reference to *Manager and is looked up
// through Manager.transcripts, guarded by m.mu — never a field on Session
// (RESEARCH Pitfall 4).
type transcriptSession struct {
	gameSessionID uuid.UUID
	userID        string
	connectionID  string

	seq     int64 // atomic; also doubles as the running line count
	dropped int64 // atomic

	muPending sync.Mutex
	pending   []byte // partial line buffer for game output

	lines chan TranscriptLine
	stop  chan struct{} // closed by close() to signal shutdown; lines is never closed (code review CR-02)
	done  chan struct{} // closed by writeLoop once it has drained and returned
}

// enqueue is the non-blocking send every line-producing call site uses:
// when the channel is full it increments the dropped counter and returns
// immediately, rather than ever blocking the MUD read/write path (T-3-26).
//
// It never sends on t.lines after close() has run: close() closes t.stop,
// never t.lines, specifically so a send here can never race a concurrent
// channel close and panic (code review CR-02). A late enqueue — one that
// arrives once shutdown has started — is dropped and counted in the same
// t.dropped counter a full channel would use, not treated as a distinct
// error.
func (t *transcriptSession) enqueue(source, text string) {
	seq := atomic.AddInt64(&t.seq, 1)

	select {
	case <-t.stop:
		atomic.AddInt64(&t.dropped, 1)
		return
	default:
	}

	select {
	case t.lines <- TranscriptLine{Seq: seq, Source: source, Text: text, CreatedAt: time.Now()}:
	case <-t.stop:
		atomic.AddInt64(&t.dropped, 1)
	default:
		atomic.AddInt64(&t.dropped, 1)
	}
}

// feedGameOutput appends p (already telnet-stripped, not yet ANSI-
// stripped) to the pending partial-line buffer, splits on '\n', trims a
// trailing '\r', and enqueues one "game" line per completed line with
// ANSI stripped (reusing plan 03-01's stripANSI), keeping any partial
// tail for the next call.
func (t *transcriptSession) feedGameOutput(p []byte) {
	if len(p) == 0 {
		return
	}

	t.muPending.Lock()
	defer t.muPending.Unlock()

	t.pending = append(t.pending, p...)
	for {
		idx := bytes.IndexByte(t.pending, '\n')
		if idx < 0 {
			break
		}
		line := bytes.TrimSuffix(t.pending[:idx], []byte("\r"))
		t.enqueue("game", string(stripANSI(line)))
		t.pending = t.pending[idx+1:]
	}

	if len(t.pending) > 0 {
		// Copy the tail so it does not retain the caller's backing array.
		tail := make([]byte, len(t.pending))
		copy(tail, t.pending)
		t.pending = tail
	} else {
		t.pending = nil
	}
}

// writeLoop drains lines from the channel, batching up to
// transcriptBatchSize lines or a transcriptBatchTick tick, and calls
// AppendGameLines for each batch. On a store error it logs one
// [AI-PLAYER] transcript ... event=write-error line carrying ids and a
// count only, never line text, and keeps draining — a database problem
// must never wedge the MUD path. Exits (closing done) once t.stop is
// closed, after a final non-blocking drain of anything already sitting in
// t.lines. t.lines itself is never closed (code review CR-02), so this
// loop never observes a closed-channel receive.
func (t *transcriptSession) writeLoop(sink TranscriptSink) {
	defer close(t.done)

	ticker := time.NewTicker(transcriptBatchTick)
	defer ticker.Stop()

	batch := make([]TranscriptLine, 0, transcriptBatchSize)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		if err := sink.AppendGameLines(t.gameSessionID, batch); err != nil {
			log.Printf("[AI-PLAYER] transcript user_id=%s connection_id=%s game_session_id=%s event=write-error count=%d",
				t.userID, t.connectionID, t.gameSessionID, len(batch))
		}
		batch = batch[:0]
	}

	for {
		select {
		case line := <-t.lines:
			batch = append(batch, line)
			if len(batch) >= transcriptBatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-t.stop:
			// Drain whatever is already buffered — anything enqueued
			// before close() started — non-blockingly, then flush and
			// exit. Anything sent after this drain loop's default case
			// fires is counted as dropped by enqueue itself.
			for {
				select {
				case line := <-t.lines:
					batch = append(batch, line)
				default:
					flush()
					return
				}
			}
		}
	}
}

// close flushes any partial trailing line, closes t.stop (never t.lines —
// code review CR-02, so a concurrent enqueue can never send on a closed
// channel and panic), waits for the write loop to drain (which flushes
// every remaining batched line), calls CloseGameSession, and logs one
// [AI-PLAYER] transcript ... event=close line carrying ids, a line count
// and a dropped count.
func (t *transcriptSession) close(sink TranscriptSink) {
	t.muPending.Lock()
	if len(t.pending) > 0 {
		final := bytes.TrimSuffix(t.pending, []byte("\r"))
		t.enqueue("game", string(stripANSI(final)))
		t.pending = nil
	}
	t.muPending.Unlock()

	close(t.stop)
	<-t.done

	if sink != nil {
		if err := sink.CloseGameSession(t.gameSessionID); err != nil {
			log.Printf("[AI-PLAYER] transcript user_id=%s connection_id=%s game_session_id=%s event=close-error",
				t.userID, t.connectionID, t.gameSessionID)
		}
	}

	log.Printf("[AI-PLAYER] transcript user_id=%s connection_id=%s game_session_id=%s event=close lines=%d dropped=%d",
		t.userID, t.connectionID, t.gameSessionID, atomic.LoadInt64(&t.seq), atomic.LoadInt64(&t.dropped))
}
