-- drop indexes (optional)
DROP INDEX IF EXISTS idx_outbox_failed;
DROP INDEX IF EXISTS idx_outbox_processing_locked;
DROP INDEX IF EXISTS idx_outbox_fetch_pending;

-- drop table
DROP TABLE IF EXISTS outbox_events;
