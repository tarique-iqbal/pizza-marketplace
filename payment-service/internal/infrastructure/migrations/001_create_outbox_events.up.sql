CREATE TABLE outbox_events (
    id               BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    aggregate_id     UUID        NOT NULL,
    event_name       VARCHAR(64) NOT NULL,
    payload          JSONB       NOT NULL,
    status           VARCHAR(16) NOT NULL DEFAULT 'pending',
    attempts         INTEGER     NOT NULL DEFAULT 0,
    locked_until     TIMESTAMPTZ,
    next_attempt_at  TIMESTAMPTZ,
    last_error       TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    processed_at     TIMESTAMPTZ,
    CONSTRAINT outbox_events_status_check
        CHECK (status IN ('pending', 'processing', 'processed', 'failed'))
);

-- core worker index
CREATE INDEX idx_outbox_fetch_pending
ON outbox_events (next_attempt_at ASC, created_at ASC)
WHERE status = 'pending';

-- lock recovery
CREATE INDEX idx_outbox_processing_locked
ON outbox_events (locked_until)
WHERE status = 'processing';

-- dead-letter / failed inspection
CREATE INDEX idx_outbox_failed
ON outbox_events (created_at DESC)
WHERE status = 'failed';
