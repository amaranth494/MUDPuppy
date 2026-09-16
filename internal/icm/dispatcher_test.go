package icm

import "testing"

// TestDispatch_AutomationPassThrough_GoesThroughSafetyChecker is the diagnostic
// test for ROADMAP criterion 4. RESEARCH.md's central finding is that a driver
// built on Engine.Process() would ship a green demo while never actually
// exercising the ICM safety checker for a plain game command: Process()
// classifies "north" as pass-through and returns before Dispatch is ever
// reached (engine.go, the pass-through early return). This test proves the
// two worlds apart: the path the driver actually uses (Dispatcher.Dispatch
// called directly) really does run the safety checker, and the tempting
// alternative (Engine.Process()) really does not.
func TestDispatch_AutomationPassThrough_GoesThroughSafetyChecker(t *testing.T) {
	t.Run("approved_when_limits_are_clear", func(t *testing.T) {
		d := NewDispatcher(NewRegistry())
		ctx := ContextAutomation

		result, icmErr := d.Dispatch(&ctx, "sess-ai-1", &NormalizedCommand{
			Command:           "north",
			Operator:          "",
			RequiresExecution: true,
		})

		// (nil, nil) is the approval signal for a plain command with no
		// registered handler in the empty operator family: checkSafety
		// passed, authority allows execution, but no handler exists for
		// "north" (there never will be one — it's a MUD command, not an
		// ICM directive). The driver reads (nil, nil) as "safety approved,
		// send it yourself" and calls manager.SendCommand next.
		if icmErr != nil {
			t.Fatalf("expected no ICM error when limits are clear, got %v", icmErr)
		}
		if result != nil {
			t.Fatalf("expected nil result for a pass-through command with no handler, got %+v", result)
		}
	})

	t.Run("refused_when_rate_limit_tripped", func(t *testing.T) {
		d := NewDispatcher(NewRegistry())
		ctx := ContextAutomation
		sessionID := "sess-ai-2"

		// Drive the safety checker past its limit the same way
		// TestDispatcher_SafetyEnforcement does: repeated RecordExecution
		// calls against one session id.
		for i := 0; i < 100; i++ {
			d.safety.RecordExecution(sessionID, "north")
		}

		// This is the subtest that makes criterion 4 falsifiable: if
		// checkSafety were skipped (e.g. because the driver called
		// Engine.Process() instead of Dispatcher.Dispatch() directly),
		// this call would be approved even though the limits are tripped.
		_, icmErr := d.Dispatch(&ctx, sessionID, &NormalizedCommand{
			Command:           "north",
			Operator:          "",
			RequiresExecution: true,
		})

		if icmErr == nil {
			t.Fatal("expected Dispatch to refuse once the safety limits are tripped")
		}
	})

	t.Run("process_does_not_reach_dispatch_for_a_plain_command", func(t *testing.T) {
		e := NewEngine()
		ctx := ContextAutomation
		sessionID := "sess-ai-3"

		// Trip the engine's own dispatcher's safety checker via the real
		// GetDispatcher() accessor the driver will use.
		for i := 0; i < 100; i++ {
			e.GetDispatcher().safety.RecordExecution(sessionID, "north")
		}

		// If Process() reached Dispatch() for this plain command, the
		// tripped safety checker above would cause resp.Error to be a
		// non-nil *ICMError (circuit breaker / rate limit refusal).
		// Observed behaviour (verified this session): Process() classifies
		// "north" as pass-through in Phase 1 (recognizer sees no reserved
		// operator, then the pass-through classifier confirms
		// ShouldPassThrough == true) and returns immediately with
		// resp.Normalized set and resp.Error nil - Dispatch is never
		// called, so the tripped safety checker has no effect at all.
		// This is the pass-through short circuit RESEARCH.md's central
		// finding warns about, and is exactly why the driver must call
		// Dispatch directly instead of relying on Process().
		resp := e.Process(&ICMRequest{
			Raw:       "north",
			Context:   ctx,
			SessionID: sessionID,
		})

		if resp == nil {
			t.Fatal("expected a non-nil response from Process")
		}
		if resp.Error != nil {
			t.Fatalf("expected Process to report no safety refusal for a pass-through command (it never reaches Dispatch), got %v", resp.Error)
		}
	})
}
