-- +migrate Up
-- Records when the user's current MUDPuppy login began (D-31, amends
-- D-10): Session Memory now lives for the MUDPuppy login, per connection
-- profile, not for the game connection. OpenGameSession
-- (internal/store/transcripts.go) reads this column back, in the same
-- INSERT that opens a new game session, to decide which earlier game
-- session's Session Memory it may inherit.
--
-- NOT NULL DEFAULT NOW() backfills every existing user at migration time --
-- deliberate: the owner is signed in on staging right now and cannot be
-- asked to sign in again, so game sessions opened after this migration
-- runs must still carry Session Memory over for him rather than starting
-- empty on a column that would otherwise begin NULL.
ALTER TABLE users
ADD COLUMN IF NOT EXISTS login_started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW();
