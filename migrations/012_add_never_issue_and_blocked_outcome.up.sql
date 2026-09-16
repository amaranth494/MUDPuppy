-- +migrate Up
-- Add the owner-editable Never-issue list to profiles (D-02) and widen the
-- ai_decisions.outcome CHECK to admit the new blocked outcome (D-06, D-08).
--
-- never_issue_list ships empty on every profile with no starter set: the
-- engine holds no game knowledge (CON-engine-has-no-game-knowledge), so a
-- pre-populated list would itself be game knowledge baked into the plumbing.
--
-- blocked joins ai_decisions' existing outcome values ('sent','refused',
-- 'failed') as a fourth, distinct value: a block is the defence working,
-- not the AI failing (D-06).
ALTER TABLE profiles
ADD COLUMN IF NOT EXISTS never_issue_list TEXT NOT NULL DEFAULT '';

-- The outcome CHECK on ai_decisions was declared inline and unnamed in
-- migration 011 (`outcome TEXT NOT NULL CHECK (outcome IN
-- ('sent','refused','failed'))`), so its real name is not knowable from the
-- source SQL alone — Postgres assigns a default constraint name that is
-- reasonable to guess but not guaranteed. A DROP CONSTRAINT IF EXISTS on a
-- guessed name would silently do nothing if the guess is wrong, leaving the
-- old three-value constraint in place and rejecting every 'blocked' row at
-- runtime. Instead, look up the constraint by its actual definition: every
-- CHECK constraint on ai_decisions whose definition mentions "outcome".
-- Migration 011 created exactly one CHECK constraint on this table (the
-- outcome one), so this loop cannot drop anything else.
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT conname
        FROM pg_constraint
        WHERE conrelid = 'ai_decisions'::regclass
          AND contype = 'c'
          AND pg_get_constraintdef(oid) LIKE '%outcome%'
    LOOP
        EXECUTE format('ALTER TABLE ai_decisions DROP CONSTRAINT %I', r.conname);
    END LOOP;
END $$;

ALTER TABLE ai_decisions
ADD CONSTRAINT ai_decisions_outcome_check CHECK (outcome IN ('sent','refused','failed','blocked'));
