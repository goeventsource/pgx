CREATE TABLE store
(
    event_id    VARCHAR(255) NOT NULL,
    event_name  VARCHAR(255) NOT NULL,
    event_data  BYTEA        NOT NULL,
    version     INT          NOT NULL CHECK (version > 0),

    stream_id   VARCHAR(255) NOT NULL,
    stream_name VARCHAR(255) NOT NULL,

    metadata    JSONB,
    occurred_at TIMESTAMPTZ  NOT NULL,
    PRIMARY KEY (stream_id, version)
);

COMMENT ON COLUMN store.stream_id IS 'Represents the identifier of a stream (for example an aggregate id).';
COMMENT ON COLUMN store.stream_name IS 'Represents the name of a stream (for example an aggregate name). This is useful to determine which aggregate this event is about.';

COMMENT ON COLUMN store.event_id IS 'Represents the identifier of an event.';
COMMENT ON COLUMN store.event_name IS 'Represents the name of an event. This is useful to determine how to deserialize the event_data.';
COMMENT ON COLUMN store.event_data IS 'Represents the domain-specific data associated with an event.';
COMMENT ON COLUMN store.version IS 'Represents the version number. This is used as locking mechanisms to ensure events are ordered and there are no conflicts in the stream.';


COMMENT ON COLUMN store.metadata IS 'Additional metadata associated with an event. Could contain information such as tracing or author of the event.';
COMMENT ON COLUMN store.occurred_at IS 'Timestamp when the event occurred.';
