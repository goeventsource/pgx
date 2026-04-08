package pgxtest

import (
	"github.com/goeventsource/goeventsource"
	"github.com/goeventsource/pgx"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SnapshotterConfig represents the configuration to create a pgx.Snapshotter via NewSnapshotter
type SnapshotterConfig[K goeventsource.ID, V goeventsource.Root[K]] struct {
	Pool      *pgxpool.Pool
	Codec     goeventsource.RootEncodeDecoder[K, V]
	Strategy  goeventsource.SnapshotterWriteStrategy[K, V]
	TableName string
}

// NewSnapshotterConfig creates a new SnapshotterConfig with default values for testing purposes.
func NewSnapshotterConfig[K goeventsource.ID, V goeventsource.Root[K]](
	pool *pgxpool.Pool,
	codec goeventsource.RootEncodeDecoder[K, V],
	strategy goeventsource.SnapshotterWriteStrategy[K, V],
) SnapshotterConfig[K, V] {
	return SnapshotterConfig[K, V]{
		Pool:      pool,
		Codec:     codec,
		Strategy:  strategy,
		TableName: "snapshots",
	}
}

// NewSnapshotter creates a new instance of a pgx.Snapshotter based on the provided StoreConfig.
func NewSnapshotter[K goeventsource.ID, V goeventsource.Root[K]](cfg SnapshotterConfig[K, V]) (*pgx.Snapshotter[K, V], error) {
	return pgx.NewSnapshotter(cfg.Pool, cfg.Codec, cfg.Strategy, cfg.TableName)
}
