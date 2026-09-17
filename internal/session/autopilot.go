package session

import (
	"errors"
	"time"
)

// AutopilotState is the wire value carried by the REST response, the
// websocket autopilot message and the browser badge class. These three
// lowercase strings are a cross-plan contract shared with plans 02-02 and
// 02-03 — do not rename or recapitalize them.
type AutopilotState string

// Autopilot state constants. Styled like manager.go's StateDisconnected/
// StateConnecting/StateConnected/StateError block.
const (
	// AutopilotOff means #AUTO OFF was the last action, or autopilot has
	// never been engaged. Also the state a server restart lands on (D-06).
	AutopilotOff AutopilotState = "off"

	// AutopilotOn means #AUTO ON engaged the switch.
	AutopilotOn AutopilotState = "on"

	// AutopilotWaiting means the switch was on when the game connection
	// dropped; it issues nothing until the connection returns (D-01).
	AutopilotWaiting AutopilotState = "waiting"
)

// AutopilotRecord is the per-user record the Manager stores in its
// autopilot map. It holds no mutex and no map itself — Manager guards
// access to the map that contains these records with its own m.mu.
type AutopilotRecord struct {
	State AutopilotState
	// ConnectionID is captured at engage time so the disconnect and resume
	// log lines can carry it, and so a resume can be refused when the next
	// session is for a different profile. It lives here, not on Session,
	// because autopilot state must outlive the session that engaged it.
	ConnectionID string
	// WaitingSince is set when the record enters waiting and cleared when
	// it leaves. Nothing in Phase 2 reads WaitingSince; it exists so that a
	// bounded-waiting-lifetime remediation, if the owner chooses one at the
	// Phase 2 security review, needs no redesign (see threat T-2-08).
	WaitingSince *time.Time
	// Epoch identifies the stint (code review CR-01 of Phase 4). It goes up
	// by one every time State becomes On -- a fresh #AUTO ON and a
	// WAITING-to-ON resume alike -- and never goes down or resets while the
	// process lives, so "the switch reads On" and "this is still the stint
	// my decision belongs to" are two different questions with two
	// different answers. The driver captures the epoch when a decision
	// starts and the Manager refuses an AI send whose epoch is no longer the
	// record's, which is what stops a decision that outlived its stint from
	// being sent in the next one after a quick re-engage.
	Epoch uint64
	// PausedByOwner and ConnectionLost are D-15's two independent waiting
	// reasons (Phase 5). A record in AutopilotWaiting always has at least
	// one of them true -- pausing while already waiting from a disconnect
	// sets PausedByOwner without changing State, and a disconnect while
	// already paused by the owner sets ConnectionLost the same way. The
	// engage hook fires only on a transition that leaves both false: a
	// reconnect alone clears only ConnectionLost, and an owner resume alone
	// clears only PausedByOwner. Both reasons are cleared whenever the
	// switch lands Off, by any cause. All bookkeeping on these two fields
	// lives in manager.go; Engage, Disengage, EnterWaiting and Resume below
	// take and return only AutopilotState and must never grow a reason
	// parameter (RESEARCH Pitfall 1).
	PausedByOwner  bool
	ConnectionLost bool
}

// ErrNoConnectedSession is returned when #AUTO ON is attempted with no
// connected game session (D-03). The switch stays off.
var ErrNoConnectedSession = errors.New("autopilot requires a connected game session")

// ErrWrongConnection is returned when #AUTO ON names a connection profile
// other than the one the live session was opened for (code review C2).
var ErrWrongConnection = errors.New("autopilot can only be engaged for the profile that is connected")

// Engage is the pure transition for #AUTO ON. off becomes on (changed
// true); on stays on (changed false, D-04 — a repeated #AUTO ON must reset
// nothing). waiting stays waiting (changed false) — this case is a
// defensive no-op, not a supported path, because the caller refuses
// engagement without a connected session (D-03), so waiting is never
// reached by this function in practice.
func Engage(cur AutopilotState) (AutopilotState, bool) {
	switch cur {
	case AutopilotOff:
		return AutopilotOn, true
	case AutopilotOn:
		return AutopilotOn, false
	case AutopilotWaiting:
		return AutopilotWaiting, false
	default:
		return cur, false
	}
}

// Disengage is the pure transition for #AUTO OFF. on becomes off (changed
// true); waiting becomes off (changed true, D-01 — #AUTO OFF while waiting
// lands on off); off stays off (changed false).
func Disengage(cur AutopilotState) (AutopilotState, bool) {
	switch cur {
	case AutopilotOn:
		return AutopilotOff, true
	case AutopilotWaiting:
		return AutopilotOff, true
	case AutopilotOff:
		return AutopilotOff, false
	default:
		return cur, false
	}
}

// EnterWaiting is the pure transition for a dropped connection. on becomes
// waiting (changed true); off stays off (changed false — an off autopilot
// is not parked by a disconnect); waiting stays waiting (changed false).
func EnterWaiting(cur AutopilotState) (AutopilotState, bool) {
	switch cur {
	case AutopilotOn:
		return AutopilotWaiting, true
	case AutopilotOff:
		return AutopilotOff, false
	case AutopilotWaiting:
		return AutopilotWaiting, false
	default:
		return cur, false
	}
}

// Resume is the pure transition for a connection returning. waiting
// becomes on (changed true); off stays off (changed false, D-01 — a
// connection returning never engages an autopilot the owner turned off);
// on stays on (changed false).
func Resume(cur AutopilotState) (AutopilotState, bool) {
	switch cur {
	case AutopilotWaiting:
		return AutopilotOn, true
	case AutopilotOff:
		return AutopilotOff, false
	case AutopilotOn:
		return AutopilotOn, false
	default:
		return cur, false
	}
}
