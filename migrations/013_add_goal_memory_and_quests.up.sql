-- +migrate Up
-- Add the session goal column to profiles (D-01), the Session Memory column
-- to game_sessions (D-10, first written by plan 04-08), and the quests
-- table (Quest Memory, D-04, D-11). Every statement is guarded so this
-- migration is safe to re-run, matching migrations 010 and 011.
--
-- session_goal ships empty on every profile: the goal is owner-typed, never
-- pre-populated, the same reasoning as never_issue_list in migration 012.
ALTER TABLE profiles
ADD COLUMN IF NOT EXISTS session_goal TEXT NOT NULL DEFAULT '';

-- session_memory is a curated bullet list (D-10), stored as a JSON array of
-- strings, living for the game connection's session and persisted with it.
ALTER TABLE game_sessions
ADD COLUMN IF NOT EXISTS session_memory JSONB NOT NULL DEFAULT '[]'::jsonb;

-- quests: one row per Quest (D-04, D-11). goal_text is the owner's typed
-- text verbatim; goal_text_normalized is the trimmed, case-folded identity
-- used for reactivate-or-create matching. status stays 'active' throughout
-- Phase 4 -- nothing in this phase closes a Quest (D-04); closure into
-- succeeded/failed/abandoned/invalidated is Phase 6's work.
CREATE TABLE IF NOT EXISTS quests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    connection_id UUID NOT NULL REFERENCES saved_connections(id) ON DELETE CASCADE,
    goal_text TEXT NOT NULL,
    goal_text_normalized TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    bullets JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- A partial unique index makes "reactivate the open Quest with a matching
-- goal, or create one" a single atomic INSERT ... ON CONFLICT, with no
-- SELECT-then-INSERT race (D-04, T-4-07). Two active Quests for the same
-- connection and the same normalized goal text can never both exist.
CREATE UNIQUE INDEX IF NOT EXISTS idx_quests_active_goal
    ON quests(connection_id, goal_text_normalized) WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_quests_connection_status ON quests(connection_id, status);
