-- +migrate Down
-- Drop conversation_lines first (child of game_sessions), then the standing-
-- coaching column, mirroring migration 013's child-before-parent drop order.
DROP TABLE IF EXISTS conversation_lines;

ALTER TABLE game_sessions
DROP COLUMN IF EXISTS coaching_suggestions;
