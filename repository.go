package pgx

import (
	"context"
	"errors"
	"fmt"

	jackc "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/goeventsource/goeventsource"
)

type (
	repoID   = goeventsource.ID
	repoRoot = goeventsource.Root[repoID]
)

var (
	// Ensure Repository implements goeventsource.Repository at compile time.
	_ goeventsource.Repository[repoID, repoRoot] = (*Repository[repoID, repoRoot])(nil)
)

// RepositoryOpt is a function signature for providing options to configure a Repository.
type RepositoryOpt[K goeventsource.ID, V goeventsource.Root[K]] func(*Repository[K, V])

// WithProjectorOpt is a RepositoryOpt that sets a slice of goeventsource.Projector for a Repository.
func WithProjectorOpt[K goeventsource.ID, V goeventsource.Root[K]](ps ...goeventsource.Projector) RepositoryOpt[K, V] {
	return func(r *Repository[K, V]) {
		r.projectors = ps
	}
}

// WithSnapshotterOpt is a RepositoryOpt that sets a snapshotter for the Repository.
func WithSnapshotterOpt[K goeventsource.ID, V goeventsource.Root[K]](s goeventsource.Snapshotter[K, V]) RepositoryOpt[K, V] {
	return func(r *Repository[K, V]) {
		r.snapshotter = s
	}
}

// Repository is a PostgreSQL goeventsource.Repository implementation.
type Repository[K goeventsource.ID, V goeventsource.Root[K]] struct {
	pool        *pgxpool.Pool
	store       goeventsource.Store[K]
	factoryFunc goeventsource.FactoryFunc[K, V]
	projectors  []goeventsource.Projector
	snapshotter goeventsource.Snapshotter[K, V]
}

// NewRepository creates a new instance of Repository.
func NewRepository[K goeventsource.ID, V goeventsource.Root[K]](
	pool *pgxpool.Pool,
	store goeventsource.Store[K],
	factoryFunc goeventsource.FactoryFunc[K, V],
	opts ...RepositoryOpt[K, V],
) *Repository[K, V] {
	r := &Repository[K, V]{
		pool:        pool,
		store:       store,
		factoryFunc: factoryFunc,
	}

	for i := range opts {
		opts[i](r)
	}

	return r
}

// Read reads the goeventsource.Events from a goeventsource.Store and rebuild the goeventsource.Root state for the given goeventsource.ID.
// It returns the root aggregate and an error if the aggregate rootID is not found or an error occurs.
// When a snapshotter is configured, Read runs snapshot load and stream load in one transaction for a consistent cut.
func (r Repository[K, V]) Read(ctx context.Context, id K) (V, error) {
	if r.snapshotter != nil {
		var result V
		err := InTransaction(ctx, r.pool, func(txConn jackc.Tx) error {
			txCtx := withValueTx(ctx, txConn)
			var readErr error
			result, readErr = r.readWithContext(txCtx, id)
			return readErr
		})
		if err != nil {
			var zero V
			return zero, err
		}
		return result, nil
	}
	return r.readWithContext(ctx, id)
}

func (r Repository[K, V]) readWithContext(ctx context.Context, id K) (V, error) {
	var (
		zero        V
		hadSnapshot bool
		root        = r.factoryFunc(id, 0)
		filter      = goeventsource.StoreStreamNoFilter()
	)

	if r.snapshotter != nil {
		snap, err := r.snapshotter.ReadSnapshot(ctx, id)
		switch {
		case errors.Is(err, goeventsource.ErrSnapshotterReadNotFound):
			// ignore
		case err != nil:
			return zero, fmt.Errorf("%w: %w", goeventsource.ErrRepositoryRead, err)
		default:
			hadSnapshot = true
			root = snap
			filter.From = goeventsource.RootVersion(root) + 1
		}
	}

	evs, err := r.store.Stream(ctx, id, filter)
	switch {
	case errors.Is(err, goeventsource.ErrStoreStreamEmpty) && hadSnapshot:
		return root, nil
	case errors.Is(err, goeventsource.ErrStoreStreamEmpty):
		return zero, fmt.Errorf("%w: %w", goeventsource.ErrRepositoryReadNotFound, err)
	case err != nil:
		return zero, fmt.Errorf("%w: %w", goeventsource.ErrRepositoryRead, err)
	default:
	}

	goeventsource.PushEvents(root, evs)

	return root, nil
}

// Write appends pending events from root to the store and runs projectors / snapshot side effects.
// It returns an error if an error occurs during the write operation.
// FlushEvents is called only after the transaction commits successfully.
func (r Repository[K, V]) Write(ctx context.Context, root V) error {
	evs := goeventsource.PeekEvents(root)
	if err := InTransaction(ctx, r.pool, func(txConn jackc.Tx) error {
		txCtx := withValueTx(ctx, txConn)
		if err := r.store.Append(txCtx, evs...); err != nil {
			return err
		}

		for i := range r.projectors {
			if err := r.projectors[i].Project(txCtx, evs...); err != nil {
				return err
			}
		}

		if r.snapshotter != nil {
			if err := r.snapshotter.WriteSnapshot(txCtx, root); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf("%w: %w", goeventsource.ErrRepositoryWrite, err)
	}

	goeventsource.FlushEvents(root)
	return nil
}
