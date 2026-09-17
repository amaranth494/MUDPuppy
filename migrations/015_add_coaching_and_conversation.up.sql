-- +migrate Up
-- Add the new standing-coaching column to game_sessions (used by plan
-- 05-06, landed here so this phase applies one migration, not two) and the
-- conversation_lines table (the AI-chatter conversation itself, D-01).
-- Every statement is guarded so this migration is safe to re-run, matching
-- migrations 011 and 013.
--
-- The new column is a curated list of standing coaching lines (D-08),
-- stored as a JSON array of strings, following session_memory's exact
-- shape and lifetime (login-scoped, D-31).
ALTER TABLE game_sessions
ADD COLUMN IF NOT EXISTS coaching_suggestions JSONB NOT NULL DEFAULT '[]'::jsonb;

-- conversation_lines: one row per line of the owner/AI-chatter conversation
-- (D-01), append-only, scoped to the game session it happened during, kept
-- beside the decisions it was about so the Logs page can show it later.
-- seq is a monotonically increasing ordering column, the same kind
-- game_session_lines' own seq column is, so the conversation can be read
-- back in the order it happened.
CREATE TABLE IF NOT EXISTS conversation_lines (
    id BIGSERIAL PRIMARY KEY,
    game_session_id UUID NOT NULL REFERENCES game_sessions(id) ON DELETE CASCADE,
    seq BIGINT NOT NULL,
    speaker TEXT NOT NULL CHECK (speaker IN ('owner','chatter','system')),
    text TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_conversation_lines_seq ON conversation_lines(game_session_id, seq);
