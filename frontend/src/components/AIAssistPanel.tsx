import { useState, useEffect, useCallback, useRef } from 'react';
import { AIDecisionPayload } from '../types';
import { getDecisions } from '../services/api';
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
 * No input element of any kind exists here (D-08) — Phase 5 adds the coaching
 * input to this same panel.
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

  useEffect(() => {
    if (!wsManager) return;
    const handleAI = (payload: AIDecisionPayload) => {
      // 04-04 D-14/D-15/D-17: every event the driver emits during a stint
      // carries the standing counts; update only the fields this payload
      // actually carries, since not every future 'ai' message (e.g. a goal
      // change) needs to.
      if (payload.calls !== undefined) setCallCount(payload.calls);
      if (payload.call_cap_set) {
        setCallCap(payload.call_cap ?? null);
      } else if (payload.calls !== undefined) {
        setCallCap(null);
      }
      if (payload.failures !== undefined) setFailureCount(payload.failures);
      if (payload.blocks !== undefined) setBlockCount(payload.blocks);
      if (payload.threshold !== undefined) setDisengageThreshold(payload.threshold);

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
        {autopilotState !== 'off' && (
          <div className="ai-assist-status-line">
            {callCap != null
              ? `Calls: ${callCount} of ${callCap}`
              : `Calls: ${callCount}`}
            {' · '}Consecutive failures: {failureCount} of {disengageThreshold}
            {' · '}Consecutive blocks: {blockCount} of {disengageThreshold}
          </div>
        )}
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
