-- +goose Up

ALTER TABLE outbox_events
    ADD COLUMN lease_expires_at TIMESTAMPTZ,
    ADD COLUMN claim_token UUID;

ALTER TABLE outbox_events
    DROP CONSTRAINT outbox_events_status_check;

ALTER TABLE outbox_events
    ADD CONSTRAINT outbox_events_status_check
    CHECK (status IN ('pending', 'processing', 'published'));

CREATE INDEX idx_outbox_events_processing_lease
    ON outbox_events(lease_expires_at)
    WHERE status = 'processing';

-- +goose Down

UPDATE outbox_events
SET status = 'pending'
WHERE status = 'processing';

DROP INDEX idx_outbox_events_processing_lease;

ALTER TABLE outbox_events
    DROP CONSTRAINT outbox_events_status_check;

ALTER TABLE outbox_events
    ADD CONSTRAINT outbox_events_status_check
    CHECK (status IN ('pending', 'published'));

ALTER TABLE outbox_events
    DROP COLUMN lease_expires_at,
    DROP COLUMN claim_token;
