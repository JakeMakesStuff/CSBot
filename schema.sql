CREATE TABLE IF NOT EXISTS webhook_logging (
    channel_id BIGINT NOT NULL,
    webhook_url TEXT NOT NULL
);

CREATE INDEX webhook_logging_channel_id ON webhook_logging (channel_id);
