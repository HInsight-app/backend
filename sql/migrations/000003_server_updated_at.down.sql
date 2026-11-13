BEGIN;

DROP INDEX IF EXISTS idx_decisions_user_server_updated;

CREATE INDEX IF NOT EXISTS idx_decisions_user_updated
    ON decisions(user_id, updated_at ASC);

ALTER TABLE decisions DROP COLUMN IF EXISTS server_updated_at;

COMMIT;
