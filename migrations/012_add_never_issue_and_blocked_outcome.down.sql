-- +migrate Down
-- Reverse migration 012: restore the three-value outcome CHECK and drop the
-- Never-issue list column.
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
