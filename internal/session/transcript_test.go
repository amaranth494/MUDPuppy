package session

import (
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

// fakeTranscriptSink is an in-memory TranscriptSink used to exercise the
// tap without a database (mirrors manager_test.go's white-box-fake
// discipline). appendFunc, when set, overrides AppendGameLines entirely —
// used by TestTranscriptDoesNotBlockReader to simulate a store call that
// never returns.
type fakeTranscriptSink struct {
	mu       sync.Mutex
	opens    []fakeOpen
	appended []TranscriptLine
	closes   []uuid.UUID

	appendFunc func(gameSessionID uuid.UUID, lines []TranscriptLine) error
}

type fakeOpen struct {
	userID       uuid.UUID
	connectionID uuid.UUID
}

func (f *fakeTranscriptSink) OpenGameSession(userID, connectionID uuid.UUID) (uuid.UUID, error) {
	f.mu.Lock()
	f.opens = append(f.opens, fakeOpen{userID: userID, connectionID: connectionID})
	f.mu.Unlock()
	return uuid.New(), nil
}

func (f *fakeTranscriptSink) AppendGameLines(gameSessionID uuid.UUID, lines []TranscriptLine) error {
	if f.appendFunc != nil {
		return f.appendFunc(gameSessionID, lines)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.appended = append(f.appended, lines...)
	return nil
}

func (f *fakeTranscriptSink) CloseGameSession(gameSessionID uuid.UUID) error {
	f.mu.Lock()
	f.closes = append(f.closes, gameSessionID)
	f.mu.Unlock()
	return nil
}

func (f *fakeTranscriptSink) opensCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.opens)
}

func (f *fakeTranscriptSink) closesCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.closes)
}

func (f *fakeTranscriptSink) allLines() []TranscriptLine {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]TranscriptLine, len(f.appended))
	copy(out, f.appended)
	return out
}

// seedTranscript drives Manager.openTranscript directly, standing in for
// what a real Connect call would do — Manager.Connect dials for real
// (manager_test.go's own seedConnectedSession comment explains why a unit
// test cannot drive a genuine connect), so this is legitimate white-box
// setup for the same reason. openTranscript takes m.mu itself (code
// review CR-01), so it must be called without the lock held.
func seedTranscript(m *Manager, userID, connID string) {
	m.openTranscript(userID, connID)
}

// closeTranscript drives Manager.closeTranscript directly, forcing a
// deterministic flush of any batched lines instead of waiting on
// transcriptBatchTick. closeTranscript takes m.mu itself (code review
// CR-01), so it must be called without the lock held.
func closeTranscript(m *Manager, userID string) {
	m.closeTranscript(userID)
}

// seedConn wires a net.Pipe as userID's connection, with a goroutine
// draining the far end so SendCommand/SendCommandAs's conn.Write never
// blocks (net.Pipe is synchronous/unbuffered). Returns the near end so the
// caller can close it during cleanup.
func seedConn(m *Manager, userID string) net.Conn {
	client, server := net.Pipe()
	m.mu.Lock()
	m.conns[userID] = client
	m.mu.Unlock()

	go func() {
		buf := make([]byte, 4096)
		for {
			if _, err := server.Read(buf); err != nil {
				return
			}
		}
	}()

	return client
}

func TestTranscriptOpensOnConnectAndClosesOnDisconnect(t *testing.T) {
	t.Run("opens_for_a_saved_profile_connection", func(t *testing.T) {
		m := newTestManager()
		sink := &fakeTranscriptSink{}
		m.SetTranscriptSink(sink)

		userID := uuid.New().String()
		connID := uuid.New().String()
		seedTranscript(m, userID, connID)

		if got := sink.opensCount(); got != 1 {
			t.Fatalf("opensCount = %d, want 1", got)
		}
	})

	t.Run("quick_connect_is_not_transcribed", func(t *testing.T) {
		m := newTestManager()
		sink := &fakeTranscriptSink{}
		m.SetTranscriptSink(sink)

		seedTranscript(m, uuid.New().String(), "") // quick connect: no profile

		if got := sink.opensCount(); got != 0 {
			t.Fatalf("opensCount = %d, want 0 (quick connect must not be transcribed, D-14)", got)
		}
	})

	t.Run("closes_on_disconnect", func(t *testing.T) {
		m := newTestManager()
		sink := &fakeTranscriptSink{}
		m.SetTranscriptSink(sink)
		userID := uuid.New().String()
		connID := uuid.New().String()
		seedConnectedSession(m, userID, connID)
		seedTranscript(m, userID, connID)

		if err := m.Disconnect(userID, ReasonUser); err != nil {
			t.Fatalf("Disconnect: %v", err)
		}

		if got := sink.closesCount(); got != 1 {
			t.Fatalf("closesCount = %d, want 1", got)
		}
	})
}

func TestTranscriptLineSources(t *testing.T) {
	m := newTestManager()
	sink := &fakeTranscriptSink{}
	m.SetTranscriptSink(sink)
	userID := uuid.New().String()
	connID := uuid.New().String()

	seedConnectedSession(m, userID, connID)
	seedTranscript(m, userID, connID)
	client := seedConn(m, userID)
	defer client.Close()

	if err := m.SendCommand(userID, "look"); err != nil {
		t.Fatalf("SendCommand: %v", err)
	}
	if err := m.SendCommandAs(userID, "north", "ai"); err != nil {
		t.Fatalf("SendCommandAs: %v", err)
	}
	m.feedTranscriptOutput(userID, []byte("\x1b[31mYou see a room.\x1b[0m\r\nExits: north.\r\n"))

	closeTranscript(m, userID)

	lines := sink.allLines()
	var human, ai, game []TranscriptLine
	for _, l := range lines {
		switch l.Source {
		case "human":
			human = append(human, l)
		case "ai":
			ai = append(ai, l)
		case "game":
			game = append(game, l)
		}
	}

	if len(human) != 1 || human[0].Text != "look" {
		t.Fatalf("human lines = %+v, want one line %q", human, "look")
	}
	if len(ai) != 1 || ai[0].Text != "north" {
		t.Fatalf("ai lines = %+v, want one line %q", ai, "north")
	}
	if len(game) != 2 {
		t.Fatalf("game lines = %+v, want 2", game)
	}
	if game[0].Text != "You see a room." {
		t.Errorf("game[0].Text = %q, want ANSI stripped %q", game[0].Text, "You see a room.")
	}
	if game[1].Text != "Exits: north." {
		t.Errorf("game[1].Text = %q, want %q (no trailing \\r)", game[1].Text, "Exits: north.")
	}
}

func TestTranscriptStintMarkers(t *testing.T) {
	m := newTestManager()
	sink := &fakeTranscriptSink{}
	m.SetTranscriptSink(sink)
	userID := uuid.New().String()
	connID := uuid.New().String()

	seedConnectedSession(m, userID, connID)
	seedTranscript(m, userID, connID)

	if _, changed, err := m.EngageAutopilot(userID, connID); err != nil || !changed {
		t.Fatalf("EngageAutopilot: changed=%v err=%v", changed, err)
	}
	if _, changed := m.DisengageAutopilot(userID, "disengage"); !changed {
		t.Fatalf("DisengageAutopilot: changed=false, want true")
	}

	closeTranscript(m, userID)

	var markers []TranscriptLine
	for _, l := range sink.allLines() {
		if l.Source == "marker" {
			markers = append(markers, l)
		}
	}

	if len(markers) != 2 {
		t.Fatalf("marker count = %d, want 2; lines=%+v", len(markers), sink.allLines())
	}
	if markers[0].Text != "[AI-ASSIST engaged]" {
		t.Errorf("markers[0].Text = %q, want %q", markers[0].Text, "[AI-ASSIST engaged]")
	}
	if markers[1].Text != "[AI-ASSIST disengaged: disengage]" {
		t.Errorf("markers[1].Text = %q, want %q", markers[1].Text, "[AI-ASSIST disengaged: disengage]")
	}
	if markers[0].Seq >= markers[1].Seq {
		t.Errorf("markers not in order: seq[0]=%d seq[1]=%d", markers[0].Seq, markers[1].Seq)
	}
}

func TestTranscriptDoesNotBlockReader(t *testing.T) {
	blockCh := make(chan struct{})
	defer close(blockCh)

	sink := &fakeTranscriptSink{
		appendFunc: func(gameSessionID uuid.UUID, lines []TranscriptLine) error {
			<-blockCh // never returns until the test closes blockCh
			return nil
		},
	}

	ts := &transcriptSession{
		gameSessionID: uuid.New(),
		userID:        "user-1",
		connectionID:  "conn-1",
		lines:         make(chan TranscriptLine, transcriptChannelCapacity),
		done:          make(chan struct{}),
	}
	go ts.writeLoop(sink)

	start := time.Now()
	const total = transcriptChannelCapacity * 3
	for i := 0; i < total; i++ {
		ts.enqueue("game", "line")
	}
	elapsed := time.Since(start)

	if elapsed > 2*time.Second {
		t.Fatalf("enqueue calls took %v for %d lines, want promptly-returning (non-blocking) sends", elapsed, total)
	}
	if got := atomic.LoadInt64(&ts.dropped); got == 0 {
		t.Fatalf("dropped = 0, want > 0 (channel should have filled while the sink's AppendGameLines blocked)")
	}
}
