-- +migrate Up
-- Create the three AI Player session tables in one pair (the orchestrator's
-- decision, 03-05-PLAN.md): game_sessions and game_session_lines (the
-- session transcript, D-14/D-15), and ai_decisions (the decision log,
-- D-12), even though ai_decisions is not written until plan 03-08. Every
-- statement is guarded so this migration is safe to re-run.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- game_sessions: one row per saved-profile game connection, from connect
-- (started_at) to disconnect (ended_at, null while still open). Quick
-- connects (no profile) never get a row here (D-14).
CREATE TABLE IF NOT EXISTS game_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    connection_id UUID NOT NULL REFERENCES saved_connections(id) ON DELETE CASCADE,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMP WITH TIME ZONE,
    line_count INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_game_sessions_connection_started ON game_sessions(connection_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_game_sessions_user ON game_sessions(user_id);

-- game_session_lines: one row per transcript line. source distinguishes
-- human-typed input, AI-issued input, game output, and autopilot
-- engage/disengage stint markers (D-14, D-15). No reasoning is ever stored
-- here (D-12 keeps reasoning in ai_decisions only).
CREATE TABLE IF NOT EXISTS game_session_lines (
    id BIGSERIAL PRIMARY KEY,
    game_session_id UUID NOT NULL REFERENCES game_sessions(id) ON DELETE CASCADE,
    seq BIGINT NOT NULL,
    source TEXT NOT NULL CHECK (source IN ('human','ai','game','marker')),
    text TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_game_session_lines_seq ON game_session_lines(game_session_id, seq);

-- ai_decisions: one row per AI decision (plan 03-08 writes this table; this
-- plan only creates it, per the orchestrator's one-migration decision).
-- window_text holds captured game text verbatim, exactly as it was shown
-- to the model, and is a Phase 3 security-review agenda item (T-3-09)
-- against safety-and-abuse-policy-v1.md's data-handling section — raised
-- here, not decided.
CREATE TABLE IF NOT EXISTS ai_decisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    connection_id UUID NOT NULL REFERENCES saved_connections(id) ON DELETE CASCADE,
    game_session_id UUID REFERENCES game_sessions(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    model_name TEXT NOT NULL DEFAULT '',
    window_text TEXT NOT NULL DEFAULT '',
    reasoning TEXT NOT NULL DEFAULT '',
    command TEXT NOT NULL DEFAULT '',
    outcome TEXT NOT NULL CHECK (outcome IN ('sent','refused','failed')),
    failure_kind TEXT NOT NULL DEFAULT '',
    notice TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_ai_decisions_connection_created ON ai_decisions(connection_id, created_at);
