package pgx

import (
	"context"
	"errors"
	"fmt"
	"time"

	jackc "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/goeventsource/goeventsource"
)

// Snapshotter is a PostgreSQL goeventsource.Snapshotter implementation.
type Snapshotter[K goeventsource.ID, V goeventsource.Root[K]] struct {
	pool       *pgxpool.Pool
	codec      goeventsource.RootEncodeDecoder[K, V]
	strategy   goeventsource.SnapshotterWriteStrategy[K, V]
	upsertStmt string
	readStmt   string
}

// NewSnapshotter creates a new instance of Snapshotter.
// tableName may be "table" or "schema.table" using letters, digits, and underscores only.
func NewSnapshotter[K goeventsource.ID, V goeventsource.Root[K]](
	pool *pgxpool.Pool,
	codec goeventsource.RootEncodeDecoder[K, V],
	strategy goeventsource.SnapshotterWriteStrategy[K, V],
	tableName string,
) (*Snapshotter[K, V], error) {
	qtable, err := sanitizeQualifiedTable(tableName)
	if err != nil {
		return nil, err
	}
	return &Snapshotter[K, V]{
		pool:     pool,
		codec:    codec,
		strategy: strategy,
		readStmt: fmt.Sprintf(`SELECT snapshot FROM %s WHERE id=$1`, qtable),
		upsertStmt: fmt.Sprintf(
			`INSERT INTO %s (id, name, version, snapshot, created_at) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, version = EXCLUDED.version, snapshot = EXCLUDED.snapshot, created_at = EXCLUDED.created_at;`,
			qtable,
		),
	}, nil
}

func (s *Snapshotter[K, V]) WriteSnapshot(ctx context.Context, root V) error {
	switch {
	case ctx.Err() != nil:
		return fmt.Errorf("%w: %w", goeventsource.ErrSnapshotterWrite, ctx.Err())
	case !s.strategy(root):
		return nil
	}

	tx, shouldCommit, err := tx(ctx, s.pool)
	if err != nil {
		return fmt.Errorf("%w: %w", goeventsource.ErrSnapshotterWrite, err)
	}

	state, err := s.codec.Encode(root)
	if err != nil {
		_ = tx.Rollback(ctx)
		return fmt.Errorf("%w: could not encode: %w", goeventsource.ErrSnapshotterWrite, err)
	}

	if _, err := tx.Exec(
		ctx,
		s.upsertStmt,
		goeventsource.RootID(root).String(),
		goeventsource.RootName(root),
		goeventsource.RootVersion(root),
		state,
		time.Now(),
	); err != nil {
		_ = tx.Rollback(ctx)
		return fmt.Errorf("%w: could not execute query: %w", goeventsource.ErrSnapshotterWrite, err)
	}

	if shouldCommit {
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("%w:  could not commit transaction: %w", goeventsource.ErrSnapshotterWrite, err)
		}
	}

	return nil
}

func (s *Snapshotter[K, V]) ReadSnapshot(ctx context.Context, k K) (V, error) {
	var (
		zero     V
		snapshot []byte
	)

	err := queryRow(ctx, s.pool, s.readStmt, k.String()).Scan(&snapshot)
	switch {
	case errors.Is(err, jackc.ErrNoRows):
		return zero, fmt.Errorf("%w: could not find: %w", goeventsource.ErrSnapshotterReadNotFound, err)
	case err != nil:
		return zero, fmt.Errorf("%w: could not execute query: %w", goeventsource.ErrSnapshotterRead, err)
	default:
	}

	root := new(V)
	if err := s.codec.Decode(snapshot, root); err != nil {
		return zero, fmt.Errorf("%w: could not decode: %w", goeventsource.ErrSnapshotterRead, err)
	}

	return *root, nil
}
