CREATE TABLE snapshots
(
    id         VARCHAR(255) NOT NULL,
    name       VARCHAR(255) NOT NULL,
    version    INT          NOT NULL CHECK (version > 0),
    snapshot   JSONB,
    created_at TIMESTAMPTZ  NOT NULL,
    PRIMARY KEY (id)
);

COMMENT ON COLUMN snapshots.id IS 'Represents the identifier of a snapshot aggregate.';
COMMENT ON COLUMN snapshots.name IS 'Represents the name of a snapshot aggregate. This is useful to determine which aggregate this snapshot is about.';
COMMENT ON COLUMN snapshots.version IS 'Represents the version number of the aggregate.';
COMMENT ON COLUMN snapshots.snapshot IS 'Represents the snapshot content.';
COMMENT ON COLUMN snapshots.created_at IS 'Timestamp when the snapshot was created.';
