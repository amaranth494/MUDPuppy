-- +migrate Down
-- Reverse migration 012: restore the three-value outcome CHECK and drop the
-- Never-issue list column.
--
-- D-26, DR-3.1-05: the older three-value CHECK (outcome IN
-- ('sent','refused','failed')) cannot accept a 'blocked' row, so any row a
-- defence layer has actually blocked must be reconciled to 'failed' before
-- that constraint is re-added below, or the ADD CONSTRAINT statement fails
-- partway through this rollback on a database that has one.
UPDATE ai_decisions SET outcome = 'failed' WHERE outcome = 'blocked';

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
ADD CONSTRAINT ai_decisions_outcome_check CHECK (outcome IN ('sent','refused','failed'));

ALTER TABLE profiles
DROP COLUMN IF EXISTS never_issue_list;
