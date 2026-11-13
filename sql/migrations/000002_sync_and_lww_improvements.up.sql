BEGIN;

-- Drop trigger on decisions so client timestamps (local_updated_at) are preserved
DROP TRIGGER IF EXISTS trg_decisions_updated_at ON decisions;

-- Add idempotency_key for write retry safety on decision creation
ALTER TABLE decisions ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(64);

-- Ensure a user cannot create duplicate decisions with the same idempotency key
CREATE UNIQUE INDEX IF NOT EXISTS idx_decisions_user_idempotency 
    ON decisions(user_id, idempotency_key) 
    WHERE idempotency_key IS NOT NULL;

-- Index to optimize sync pull by updated_at
CREATE INDEX IF NOT EXISTS idx_decisions_user_updated 
    ON decisions(user_id, updated_at ASC);

COMMIT;
