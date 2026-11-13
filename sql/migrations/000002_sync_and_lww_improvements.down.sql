BEGIN;

DROP INDEX IF EXISTS idx_decisions_user_updated;
DROP INDEX IF EXISTS idx_decisions_user_idempotency;
ALTER TABLE decisions DROP COLUMN IF EXISTS idempotency_key;

CREATE TRIGGER trg_decisions_updated_at
    BEFORE UPDATE ON decisions
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMIT;
