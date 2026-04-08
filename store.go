package pgx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/goeventsource/goeventsource"
)

var (
	// Ensure Store implements goeventsource.Store at compile time
	_ goeventsource.Store[goeventsource.ID] = &Store[goeventsource.ID]{}
)

// Store is a PostgreSQL implementation of the goeventsource.Store interface.
type Store[K goeventsource.ID] struct {
	pool           *pgxpool.Pool
	codec          goeventsource.DomainEventEncodeDecoder
	appendStmt     string
	streamStmt     string
	primaryKeyName string // Postgres unique-violation constraint name for version conflicts (see isConflictError).
	appendOpts     []goeventsource.StoreAppendOpt
}

// NewStore creates a new instance of the Store.
// tableName may be "table" or "schema.table" using letters, digits, and underscores only.
func NewStore[K goeventsource.ID](
	pool *pgxpool.Pool,
	codec goeventsource.DomainEventEncodeDecoder,
	tableName string,
	primaryKeyName string,
	opts ...goeventsource.StoreAppendOpt,
) (*Store[K], error) {
	qtable, err := sanitizeQualifiedTable(tableName)
	if err != nil {
		return nil, err
	}
	if err := validateSQLIdentifier(primaryKeyName); err != nil {
		return nil, fmt.Errorf("primaryKeyName (Postgres constraint name for version conflicts): %w", err)
	}
	return &Store[K]{
		pool:   pool,
		codec:  codec,
		appendStmt: fmt.Sprintf(
			`INSERT INTO %s (event_id, event_name, event_data, version, stream_id, stream_name, metadata, occurred_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			qtable,
		),
		streamStmt: fmt.Sprintf(
			`SELECT event_id, event_name, event_data, version, stream_id, stream_name, metadata, occurred_at FROM %s WHERE stream_id = $1 AND version >= $2 ORDER BY version`,
			qtable,
		),
		primaryKeyName: primaryKeyName,
		appendOpts:     opts,
	}, nil
}

// Append appends the given goeventsource.Event to the Store.
func (s *Store[K]) Append(ctx context.Context, evs ...goeventsource.Event) error {
	tx, shouldCommit, err := tx(ctx, s.pool)
	if err != nil {
		return fmt.Errorf("%w: %w", goeventsource.ErrStoreAppend, err)
	}

	for i := range evs {
		ev := evs[i]
		for _, opt := range s.appendOpts {
			ev.Metadata = opt(ctx, ev.Metadata)
		}

		data, err := s.codec.Encode(ev)
		if err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("%w: could not encode domain event: %w", goeventsource.ErrStoreAppend, err)
		}

		metadata, err := json.Marshal(ev.Metadata)
		if err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("%w: could not encode metadata: %w", goeventsource.ErrStoreAppend, err)
		}

		if _, err := tx.Exec(
			ctx,
			s.appendStmt,
			ev.ID.String(),
			ev.DomainEventName,
			data,
			ev.Version,
			ev.StreamID.String(),
			ev.StreamName,
			metadata,
			ev.OccurredAt,
		); err != nil {
			_ = tx.Rollback(ctx)
			switch {
			case s.isConflictError(err):
				return fmt.Errorf("%w: could not commit appended events: %w", goeventsource.ErrStoreAppendVersionConflict, err)
			default:
				return fmt.Errorf("%w: could not insert event model: %w", goeventsource.ErrStoreAppend, err)
			}
		}
	}

	if shouldCommit {
		switch err := tx.Commit(ctx); {
		case s.isConflictError(err):
			return fmt.Errorf("%w: could not commit appended events: %w", goeventsource.ErrStoreAppendVersionConflict, err)
		case err != nil:
			return fmt.Errorf("%w: could not commit appended events: %w", goeventsource.ErrStoreAppend, err)
		}
	}

	return nil
}

// Stream retrieves the goeventsource.Events associated with the given goeventsource.ID from the Store.
func (s *Store[K]) Stream(ctx context.Context, id K, f goeventsource.StoreStreamFilter) (goeventsource.Events, error) {
	rows, err := query(ctx, s.pool, s.streamStmt, id.String(), f.From)
	if err != nil {
		return nil, fmt.Errorf("%w: could not list events: %w", goeventsource.ErrStoreStream, err)
	}
	defer rows.Close()

	var evs goeventsource.Events
	for rows.Next() {
		var (
			rowID     simpleID
			streamID  simpleID
			eventData []byte
			metadata  []byte
			ev        goeventsource.Event
		)

		if err := rows.Scan(
			&rowID,
			&ev.DomainEventName,
			&eventData,
			&ev.Version,
			&streamID,
			&ev.StreamName,
			&metadata,
			&ev.OccurredAt,
		); err != nil {
			return nil, fmt.Errorf("%w: could not scan event: %w", goeventsource.ErrStoreStream, err)
		}

		ev.ID = rowID
		ev.StreamID = streamID
		if err := s.codec.Decode(eventData, &ev); err != nil {
			return nil, fmt.Errorf("%w: could not decode domain events: %w", goeventsource.ErrStoreStream, err)
		}
		if err := json.Unmarshal(metadata, &ev.Metadata); err != nil {
			return nil, fmt.Errorf("%w: could not decode metadata: %w", goeventsource.ErrStoreStream, err)
		}
		evs = append(evs, ev)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: could not list events: %w", goeventsource.ErrStoreStream, err)
	}

	if len(evs) == 0 {
		return nil, goeventsource.ErrStoreStreamEmpty
	}

	return evs, nil
}

func (s *Store[K]) isConflictError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505" && pgErr.ConstraintName == s.primaryKeyName
	}

	return false
}
