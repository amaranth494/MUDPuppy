import { useState, useEffect, useCallback } from 'react';
import { useParams } from 'react-router-dom';
import { GameSessionSummary, TranscriptLine, ChatLine } from '../types';
import { getGameSessions, getGameSessionTranscript, getConnection, getSessionConversation } from '../services/api';
import Header from '../components/Header';

/**
 * LogsPage — the per-profile log page (D-17), opened in its own browser tab by
 * the Settings "Open Logging" button (D-16). A left pane lists the profile's
 * recorded sessions by date and time; the right pane shows the selected
 * session's transcript with human, AI and game lines told apart.
 *
 * This is a plain SPA route rendered inside the app's global AuthGuard
 * (App.tsx) — sign-in protection comes free from that wrapper, with zero new
 * auth code on this page. The only connection id this page ever uses comes
 * from the route params, never from any other profile the caller might try
 * to smuggle in.
 *
 * This page only reads: nothing here removes data, saves a copy to disk,
 * searches across sessions, or lists another profile's sessions (D-17).
 */
export default function LogsPage() {
  const { connectionId } = useParams<{ connectionId: string }>();

  const [profileName, setProfileName] = useState<string>('');
  const [sessions, setSessions] = useState<GameSessionSummary[]>([]);
  const [sessionsLoading, setSessionsLoading] = useState(true);
  const [sessionsError, setSessionsError] = useState(false);

  const [selectedSessionId, setSelectedSessionId] = useState<string | null>(null);
  const [transcript, setTranscript] = useState<TranscriptLine[] | null>(null);
  const [transcriptLoading, setTranscriptLoading] = useState(false);
  const [transcriptError, setTranscriptError] = useState(false);

  // The coaching conversation section (D-04, D-27): its own read, its own
  // loading/error state, kept entirely separate from the transcript above —
  // never merged and never interleaved by time (05-UI-SPEC.md §6).
  const [conversation, setConversation] = useState<ChatLine[] | null>(null);
  const [conversationLoading, setConversationLoading] = useState(false);
  const [conversationError, setConversationError] = useState(false);

  // Load the profile's display name for the page heading and browser tab
  // title. getConnection(id) is the existing per-connection lookup every
  // other screen already uses (SettingsPage.tsx); if it fails for any reason
  // the connection id itself stands in for the name rather than adding a new
  // endpoint just for this page's heading.
  useEffect(() => {
    if (!connectionId) return;
    getConnection(connectionId)
      .then((conn) => setProfileName(conn.name))
      .catch(() => setProfileName(connectionId));
  }, [connectionId]);

  useEffect(() => {
    document.title = `Session Logs — ${profileName || connectionId || ''}`;
  }, [profileName, connectionId]);

  const loadSessions = useCallback(async () => {
    if (!connectionId) return;
    setSessionsLoading(true);
    setSessionsError(false);
    try {
      const data = await getGameSessions(connectionId);
      setSessions(data);
    } catch {
      setSessionsError(true);
    } finally {
      setSessionsLoading(false);
    }
  }, [connectionId]);

  useEffect(() => {
    loadSessions();
  }, [loadSessions]);

  const handleSelectSession = useCallback(
    async (sessionId: string) => {
      if (!connectionId) return;
      setSelectedSessionId(sessionId);
      setTranscriptLoading(true);
      setTranscriptError(false);
      setTranscript(null);
      try {
        const lines = await getGameSessionTranscript(connectionId, sessionId);
        setTranscript(lines);
      } catch {
        setTranscriptError(true);
      } finally {
        setTranscriptLoading(false);
      }
    },
    [connectionId]
  );

  // The conversation section's own fetch-on-select (D-04, D-27), in the same
  // shape as the transcript's own fetch-on-select above, but as its own
  // effect keyed on the selected session so it never blocks or is blocked by
  // the transcript's own load state.
  useEffect(() => {
    if (!connectionId || !selectedSessionId) {
      setConversation(null);
      setConversationError(false);
      return;
    }
    let cancelled = false;
    setConversationLoading(true);
    setConversationError(false);
    setConversation(null);
    getSessionConversation(connectionId, selectedSessionId)
      .then((lines) => {
        if (!cancelled) setConversation(lines);
      })
      .catch(() => {
        if (!cancelled) setConversationError(true);
      })
      .finally(() => {
        if (!cancelled) setConversationLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [connectionId, selectedSessionId]);

  return (
    <>
      <Header />
      <div className="logs-page">
        <div className="logs-page-header">
          <h2>Session Logs — {profileName || connectionId}</h2>
        </div>
        <div className="logs-layout">
          <div className="logs-session-list">
            {sessionsLoading && (
              <div className="logs-session-empty">Loading sessions…</div>
            )}
            {!sessionsLoading && sessionsError && (
              <div className="logs-session-empty">
                Failed to load sessions — refresh the page to try again
              </div>
            )}
            {!sessionsLoading && !sessionsError && sessions.length === 0 && (
              <div className="logs-session-empty">
                No sessions recorded yet for this profile.
              </div>
            )}
            {!sessionsLoading &&
              !sessionsError &&
              sessions.map((session) => (
                <div
                  key={session.id}
                  className={`logs-session-row${selectedSessionId === session.id ? ' active' : ''}`}
                  onClick={() => handleSelectSession(session.id)}
                >
                  Session {formatSessionLabel(session.started_at)}
                </div>
              ))}
          </div>
          {/* The right-hand column (05-UI-SPEC.md §6): a plain wrapper
              carrying only layout properties inline, so .logs-transcript
              itself is untouched (byte-identical to before this plan,
              per this plan's own acceptance criteria) and index.css --
              owned by a parallel plan's own work this phase -- is never
              reopened. The conversation section becomes a true sibling of
              .logs-transcript, below it, never inside it. */}
          <div style={{ flex: 1, display: 'flex', flexDirection: 'column', minHeight: 0 }}>
          <div className="logs-transcript">
            {!selectedSessionId && !transcriptLoading && (
              <div className="logs-transcript-empty">
                Select a session on the left to view its transcript.
              </div>
            )}
            {selectedSessionId && transcriptLoading && (
              <div className="logs-transcript-empty">Loading transcript…</div>
            )}
            {selectedSessionId && !transcriptLoading && transcriptError && (
              <div className="logs-transcript-empty">
                Failed to load transcript — refresh the page to try again
              </div>
            )}
            {selectedSessionId &&
              !transcriptLoading &&
              !transcriptError &&
              transcript &&
              transcript.map((line) => (
                <div key={line.seq} className={`logs-transcript-line ${line.source}`}>
                  {renderTranscriptLine(line)}
                </div>
              ))}
          </div>
          {/* The coaching conversation section (D-04, D-27, 05-UI-SPEC.md
              §6): its own labelled block below the transcript pane, never
              woven into it. Only rendered once a session is selected --
              the transcript's own "select a session" empty state above
              already covers the nothing-selected case for this pane. */}
          {selectedSessionId && (
            <div className="logs-conversation" style={{ overflowY: 'auto' }}>
              <h3>Coaching Conversation</h3>
              {conversationLoading && (
                <div className="logs-conversation-empty">Loading conversation…</div>
              )}
              {!conversationLoading && conversationError && (
                <div className="logs-conversation-empty">
                  Failed to load conversation — refresh the page to try again
                </div>
              )}
              {!conversationLoading &&
                !conversationError &&
                conversation &&
                (conversation.length === 0 ? (
                  <div className="logs-conversation-empty">
                    No coaching conversation recorded for this session.
                  </div>
                ) : (
                  conversation.map((line) => (
                    <div
                      key={line.id}
                      className={`logs-conversation-line ${line.speaker === 'owner' ? 'human' : 'chatter'}`}
                    >
                      {line.speaker === 'owner' ? `You: ${line.text}` : `AI-chatter: ${line.text}`}
                    </div>
                  ))
                ))}
            </div>
          )}
          </div>
        </div>
      </div>
    </>
  );
}

// renderTranscriptLine applies the one canonical prefix per source (D-14):
// human lines get "> ", AI lines get the identical AI-ASSIST label the live
// terminal and panel already use, game and marker lines get no prefix at
// all. Rendered as a plain JSX string child so React escapes it — a
// transcript is untrusted remote text by definition (T-3-37).
function renderTranscriptLine(line: TranscriptLine): string {
  switch (line.source) {
    case 'human':
      return `> ${line.text}`;
    case 'ai':
      return `[AI-ASSIST > ${line.text}]`;
    default:
      return line.text;
  }
}

// formatSessionLabel renders a session's started_at as YYYY-MM-DD HH:MM:SS in
// 24-hour form (Claude's Discretion, 03-UI-SPEC.md section 5) — local time,
// since that is what the owner reads on their own clock.
function formatSessionLabel(startedAt: string): string {
  const d = new Date(startedAt);
  if (isNaN(d.getTime())) return startedAt;
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}
