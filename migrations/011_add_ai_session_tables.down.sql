-- +migrate Down
-- Drop the three AI Player session tables, children before parents.
DROP TABLE IF EXISTS ai_decisions;
DROP TABLE IF EXISTS game_session_lines;
DROP TABLE IF EXISTS game_sessions;
