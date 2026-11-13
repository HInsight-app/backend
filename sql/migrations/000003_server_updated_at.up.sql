BEGIN;

-- updated_at holds the client's edit time (used for last-write-wins).
-- server_updated_at holds the server's time of the last write to the decision
-- or any of its child rows (options, outcome, evaluation, tags). Pull
-- (GET /decisions?updated_since=...) filters on it, so an edit made offline
-- earlier but synced later is still picked up by other devices.
ALTER TABLE decisions
    ADD COLUMN IF NOT EXISTS server_updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT clock_timestamp();

DROP INDEX IF EXISTS idx_decisions_user_updated;

CREATE INDEX IF NOT EXISTS idx_decisions_user_server_updated
    ON decisions(user_id, server_updated_at ASC);

COMMIT;
