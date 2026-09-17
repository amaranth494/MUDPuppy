import { useState, useEffect, useCallback, useRef } from 'react';
import { AIDecisionPayload } from '../types';
import { getDecisions, getGoal, putGoal, getSessionMemory } from '../services/api';
import { useSession } from '../context/SessionContext';

interface AIAssistPanelProps {
  connectionId: string;
}

// A decision card (sent, or a reload/live failure whose command slot carries the
// stored/pushed notice text instead) — see 03-UI-SPEC.md's reload paragraph.
interface DecisionEntry {
  kind: 'decision';
  id: string;
  reasoning: string;
  command: string;
  outcome: string;
  reason?: string;
}

// A live-only failure/refusal notice pushed as kind === 'system' on the 'ai'
// websocket message (plan 03-09). Never replayed after a reload (03-UI-SPEC.md).
interface SystemEntry {
  kind: 'system';
  id: string;
  message: string;
  outcome: string;
}

type PanelEntry = DecisionEntry | SystemEntry;

/**
 * AIAssistPanel — the floating, minimizable AI Assist panel (D-06 to D-08).
 *
 * Reloads the connection's decision history once on mount via getDecisions, then
 * stays live via the wsManager's onAI/offAI pair (plan 03-09's 'ai' websocket
 * message). The minimized tab's state dot reads autopilotState straight from
 * SessionContext, the same "server owns truth, derive nothing" discipline
 * AutopilotBadge.tsx uses — this component never polls and never becomes a second
 * opinion about the switch's position.
 *
 * The goal box (D-01 to D-04, plan 04-06) is the panel's first exception
 * to "no input element" (D-08): a single text input, saved on blur/Enter
 * with no Save button, kept deliberately minimal. Phase 5 adds the
 * coaching input to this same panel.
 */
export default function AIAssistPanel({ connectionId }: AIAssistPanelProps) {
  const { wsManager, autopilotState } = useSession();
  const [collapsed, setCollapsed] = useState(false);
  const [entries, setEntries] = useState<PanelEntry[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const bodyRef = useRef<HTMLDivElement>(null);

  // 04-04: the standing status line's counts (D-14, D-15, D-17), held in
  // component state and updated from whichever fields the latest 'ai'
  // message actually carries (04-UI-SPEC.md §3's Claude's Discretion) --
  // never assumed present on every message, since only the driver's own
  // events populate them. callCap is null when the stint has no cap set.
  const [callCount, setCallCount] = useState(0);
  const [callCap, setCallCap] = useState<number | null>(null);
  const [failureCount, setFailureCount] = useState(0);
  const [blockCount, setBlockCount] = useState(0);
  const [disengageThreshold, setDisengageThreshold] = useState(3);

  // 04-06: the goal box (D-01 to D-04). goalLoadedRef guards
  // commitGoal against firing before the initial getGoal load completes
  // (a stray blur before load must never overwrite the stored goal with an
  // empty string). lastCommittedGoalRef holds the last value actually
  // saved, so committing the same text twice (e.g. blur without an edit)
  // is a no-op — no wasted PUT, no spurious system line.
  const [goal, setGoal] = useState('');
  const goalLoadedRef = useRef(false);
  const lastCommittedGoalRef = useRef('');
  // goalRef always holds what the box holds right now, so a save loop that
  // outlives the render it started in still reads the latest typing;
  // savingGoalRef is the one-save-in-flight guard (code review WR-08).
  const goalRef = useRef('');
  const savingGoalRef = useRef(false);

  // 04-08: the read-only, collapsible Session Memory section (D-10).
  // memoryOpen defaults to collapsed and is component-only state — it
  // resets on remount, never persisted (04-UI-SPEC.md Claude's Discretion).
  // sessionMemory loads once on mount via getSessionMemory and is then
  // replaced wholesale from every 'ai' message that carries a
  // session_memory snapshot (never merged, never diffed).
  const [memoryOpen, setMemoryOpen] = useState(false);
  const [sessionMemory, setSessionMemory] = useState<string[]>([]);

  // The counts are meaningless before an engage and reset to zero on every
  // new stint (D-14/D-15/D-17); the status line itself is hidden entirely
  // while autopilot is off (04-UI-SPEC.md §3), but resetting here too keeps
  // a lingering high count from flashing briefly the moment it re-shows.
  useEffect(() => {
    if (autopilotState === 'off') {
      setCallCount(0);
      setFailureCount(0);
      setBlockCount(0);
    }
  }, [autopilotState]);

  const loadHistory = useCallback(async () => {
    if (!connectionId) return;
    setIsLoading(true);
    try {
      const stored = await getDecisions(connectionId);
      const mapped: PanelEntry[] = stored.map((item) => ({
        kind: 'decision',
        id: item.id,
        reasoning: item.reasoning,
        // 03-UI-SPEC.md's reload paragraph: a failed/refused row's stored notice
        // is the visible text, carried in the command slot so the outcome-failed/
        // outcome-refused CSS override (red) applies to it like any other card.
        // A blocked row keeps the real command in this slot too (03.1-UI-SPEC.md,
        // RESEARCH Pitfall 5) — only failed/refused collapse the command into the notice.
        command: item.outcome === 'failed' || item.outcome === 'refused' ? item.notice : item.command,
        reason: item.outcome === 'blocked' ? item.notice : undefined,
        outcome: item.outcome,
      }));
      setEntries(mapped);
    } catch {
      setEntries([]);
    } finally {
      setIsLoading(false);
    }
  }, [connectionId]);

  useEffect(() => {
    loadHistory();
  }, [loadHistory]);

  // Load the current goal on mount and whenever the connection changes
  // (D-01: it survives a refresh and a reconnect). A load failure leaves
  // the box at its default empty value (the empty-goal placeholder is the
  // only empty-state rendering — D-02) rather than blocking the rest of
  // the panel.
  useEffect(() => {
    if (!connectionId) return;
    let cancelled = false;
    goalLoadedRef.current = false;
    getGoal(connectionId)
      .then((resp) => {
        if (cancelled) return;
        setGoal(resp.goal);
        goalRef.current = resp.goal;
        lastCommittedGoalRef.current = resp.goal;
      })
      .catch(() => {
        // No panel-wide error surface exists for a load failure elsewhere
        // in this component either (loadHistory above has the same
        // silent-fallback shape); the box simply stays at its default.
      })
      .finally(() => {
        if (!cancelled) goalLoadedRef.current = true;
      });
    return () => {
      cancelled = true;
    };
  }, [connectionId]);

  // Load the current Session Memory once on mount and whenever the
  // connection changes (D-10). A load failure leaves the section at its
  // default empty state — same silent-fallback shape as the goal and
  // history loads above — rather than blocking the rest of the panel.
  useEffect(() => {
    if (!connectionId) return;
    let cancelled = false;
    getSessionMemory(connectionId)
      .then((resp) => {
        if (cancelled) return;
        setSessionMemory(resp.session_memory);
      })
      .catch(() => {
        // No panel-wide error surface for a load failure here either.
      });
    return () => {
      cancelled = true;
    };
  }, [connectionId]);

  // Commits on blur and on Enter (which blurs the input) — no Save button
  // (04-UI-SPEC.md Layout §1). A failed save never reverts or clears the
  // owner's typing; it is surfaced through the same system-line mechanism
  // every other failure in this panel already uses.
  //
  // Saves are serialised (code review WR-08): at most one PUT is in flight.
  // Blur (PUT A), a quick edit, then Enter (PUT B) used to run concurrently;
  // the server could apply them in either order and lastCommittedGoalRef was
  // set by whichever response arrived LAST, so the box could show B while
  // the server held A and the next blur was a no-op. Now a commit that
  // arrives while one is in flight just returns: when the in-flight save
  // finishes, the loop below looks at what the box holds NOW (goalRef) and
  // saves again if it differs — the last typed value always wins.
  const commitGoal = useCallback(async () => {
    if (!connectionId || !goalLoadedRef.current) return;
    if (savingGoalRef.current) return;
    savingGoalRef.current = true;
    try {
      while (goalRef.current !== lastCommittedGoalRef.current) {
        const toSave = goalRef.current;
        try {
          await putGoal(connectionId, { goal: toSave });
          lastCommittedGoalRef.current = toSave;
        } catch {
          setEntries((prev) => [
            ...prev,
            {
              kind: 'system',
              id: `goal-save-failed-${Date.now()}`,
              message: '[Failed to save session goal — try again]',
              outcome: 'failed',
            },
          ]);
          // Leave lastCommittedGoalRef alone so the next blur/Enter retries,
          // and stop looping: a failing server must not be hammered.
          break;
        }
      }
    } finally {
      savingGoalRef.current = false;
    }
  }, [connectionId]);

  useEffect(() => {
    if (!wsManager) return;
    const handleAI = (payload: AIDecisionPayload) => {
      // 04-04 D-14/D-15/D-17: every event the driver emits during a stint
      // carries the standing counts; update only the fields this payload
      // actually carries, since not every 'ai' message (e.g. a goal change)
      // does. Code review WR-05: the server now sends these fields on every
      // stint message even when they are 0 (it used to leave a zero out, so
      // "Consecutive failures: 1 of 3" stayed on screen after the streak had
      // been cleared). "Carried" therefore means "is a number", and a
      // carried 0 is applied like any other value.
      const carried = (v: unknown): v is number => typeof v === 'number';
      if (carried(payload.calls)) setCallCount(payload.calls);
      if (payload.call_cap_set) {
        setCallCap(payload.call_cap ?? null);
      } else if (carried(payload.calls)) {
        setCallCap(null);
      }
      if (carried(payload.failures)) setFailureCount(payload.failures);
      if (carried(payload.blocks)) setBlockCount(payload.blocks);
      if (carried(payload.threshold)) setDisengageThreshold(payload.threshold);
      // 04-08 D-10: the full curated list rides every stint message —
      // replace wholesale, never merge or diff. An EMPTY list is a real
      // value too (the AI emptied its memory) and replaces the old bullets.
      if (Array.isArray(payload.session_memory)) setSessionMemory(payload.session_memory);

      if (payload.kind === 'decision') {
        setEntries((prev) => [
          ...prev,
          {
            kind: 'decision',
            id: payload.id,
            reasoning: payload.reasoning ?? '',
            command: payload.command ?? '',
            outcome: payload.outcome ?? 'sent',
            reason: payload.outcome === 'blocked' ? payload.message : undefined,
          },
        ]);
      } else {
        setEntries((prev) => [
          ...prev,
          {
            kind: 'system',
            id: payload.id,
            // One canonical string per event: the same bracketed notice the
            // terminal writes (03-UI-SPEC.md's Copywriting Contract).
            message: `[${payload.message ?? ''}]`,
            outcome: payload.outcome ?? 'failed',
          },
        ]);
      }
    };
    wsManager.onAI(handleAI);
    return () => {
      wsManager.offAI(handleAI);
    };
  }, [wsManager]);

  // Keep the list scrolled to the newest entry.
  useEffect(() => {
    if (bodyRef.current) {
      bodyRef.current.scrollTop = bodyRef.current.scrollHeight;
    }
  }, [entries]);

  if (collapsed) {
    return (
      <button className={`ai-assist-tab state-${autopilotState}`} onClick={() => setCollapsed(false)}>
        <span className="ai-assist-tab-dot" />
        <span>🤖 AI Assist</span>
      </button>
    );
  }

  return (
    <div className="ai-assist-panel">
      <div className="ai-assist-panel-header">
        <span className="ai-assist-panel-title">AI Assist</span>
        <button
          className="ai-assist-panel-minimize"
          onClick={() => setCollapsed(true)}
          title="Minimize"
          aria-label="Minimize"
        >
          –
        </button>
      </div>
      <div className="ai-assist-panel-top">
        <div className="form-group">
          <label className="form-label">Session Goal</label>
          <input
            type="text"
            className="form-input"
            placeholder="No session goal set"
            value={goal}
            onChange={(e) => {
              goalRef.current = e.target.value;
              setGoal(e.target.value);
            }}
            onBlur={commitGoal}
            onKeyDown={(e) => {
              if (e.key === 'Enter') (e.target as HTMLInputElement).blur();
            }}
          />
          <p className="form-hint">
            Editable anytime, even while the AI plays. The next decision picks up the new goal.
          </p>
        </div>
        {autopilotState !== 'off' && (
          <div className="ai-assist-status-line">
            {callCap != null
              ? `Calls: ${callCount} of ${callCap}`
              : `Calls: ${callCount}`}
            {' · '}Consecutive failures: {failureCount} of {disengageThreshold}
            {' · '}Consecutive blocks: {blockCount} of {disengageThreshold}
          </div>
        )}
        <div className="ai-assist-memory">
          <button className="ai-assist-memory-header" onClick={() => setMemoryOpen((v) => !v)}>
            <span>{memoryOpen ? '▾' : '▸'}</span>
            <span>Session Memory ({sessionMemory.length})</span>
          </button>
          {memoryOpen &&
            (sessionMemory.length === 0 ? (
              <div className="ai-assist-memory-empty">No session memory yet.</div>
            ) : (
              <ul className="ai-assist-memory-list">
                {sessionMemory.map((item, i) => (
                  <li key={i} className="ai-assist-memory-item">
                    {item}
                  </li>
                ))}
              </ul>
            ))}
        </div>
      </div>
      <div className="ai-assist-panel-body" ref={bodyRef}>
        {isLoading && <div className="ai-assist-loading">Loading decision history…</div>}
        {!isLoading && entries.length === 0 && (
          <div className="ai-assist-empty">
            No AI decisions yet. Engage autopilot with #AUTO ON to see the AI's reasoning here.
          </div>
        )}
        {entries.map((entry, index) => {
          // Live-pushed system entries can carry an empty id (Driver.recordFailure
          // sets decisionID = "" when the decision store is nil or the insert
          // fails), and two such entries in the same render would collide on
          // key={entry.id}. Fall back to the entry's stable array index — entries
          // are only ever appended, never reordered or removed, so the index is a
          // stable per-entry key for the lifetime of this component instance.
          const key = entry.id || `no-id-${index}`;
          return entry.kind === 'decision' ? (
            <div key={key} className={`ai-decision outcome-${entry.outcome}`}>
              <div className="ai-decision-reasoning">{entry.reasoning}</div>
              <div className="ai-decision-command">
                {entry.outcome === 'blocked' ? `Blocked → ${entry.command}` : `→ ${entry.command}`}
              </div>
              {entry.outcome === 'blocked' && entry.reason && (
                <div className="ai-decision-blocked-reason">{entry.reason}</div>
              )}
            </div>
          ) : (
            <div key={key} className={`ai-system-line state-${entry.outcome}`}>
              {entry.message}
            </div>
          );
        })}
      </div>
    </div>
  );
}
