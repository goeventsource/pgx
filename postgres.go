package pgx

import (
	"context"
	"fmt"
	"strings"

	jackc "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// validateSQLIdentifier rejects characters that must never appear in unquoted Postgres identifiers
// we interpolate into SQL (defense in depth; prefer static table names in application code).
func validateSQLIdentifier(s string) error {
	if s == "" {
		return fmt.Errorf("empty identifier")
	}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			continue
		}
		return fmt.Errorf("invalid character %q in SQL identifier %q", r, s)
	}
	return nil
}

const maxQualifiedTableParts = 2

// sanitizeQualifiedTable builds a quoted "schema"."table" or "table" fragment using pgx.Identifier.
func sanitizeQualifiedTable(qual string) (string, error) {
	parts := strings.Split(qual, ".")
	if len(parts) > maxQualifiedTableParts {
		return "", fmt.Errorf("table name %q: use at most schema.table", qual)
	}
	id := make(jackc.Identifier, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if err := validateSQLIdentifier(p); err != nil {
			return "", fmt.Errorf("table name %q: %w", qual, err)
		}
		id = append(id, p)
	}
	return id.Sanitize(), nil
}

// simpleID represents a basic implementation of a goeventsource.ID
type simpleID string

// String returns the string representation of the simpleID.
func (id simpleID) String() string {
	return string(id)
}

type txValue struct{}

// withValueTx returns a context that carries the given jackc/pgx transaction.
func withValueTx(ctx context.Context, tx jackc.Tx) context.Context {
	return context.WithValue(ctx, txValue{}, tx)
}

// valueTx returns the jackc/pgx transaction from the context, if any.
func valueTx(ctx context.Context) (jackc.Tx, bool) {
	tx, ok := ctx.Value(txValue{}).(jackc.Tx)
	if !ok {
		return nil, false
	}
	return tx, true
}

// queryRow runs QueryRow on the transaction from ctx when present, otherwise on pool.
func queryRow(ctx context.Context, pool *pgxpool.Pool, sql string, args ...any) jackc.Row {
	if t, ok := valueTx(ctx); ok {
		return t.QueryRow(ctx, sql, args...)
	}
	return pool.QueryRow(ctx, sql, args...)
}

// query runs Query on the transaction from ctx when present, otherwise on pool.
func query(ctx context.Context, pool *pgxpool.Pool, sql string, args ...any) (jackc.Rows, error) {
	if t, ok := valueTx(ctx); ok {
		return t.Query(ctx, sql, args...)
	}
	return pool.Query(ctx, sql, args...)
}

// InTransaction runs fn with a pgx transaction. If ctx already carries a tx (via withValueTx),
// fn runs in that transaction and this function does not commit or roll back. Otherwise it begins a
// new transaction, runs fn, commits on success or rolls back on failure.
func InTransaction(ctx context.Context, pool *pgxpool.Pool, fn func(jackc.Tx) error) error {
	if t, ok := valueTx(ctx); ok {
		return fn(t)
	}
	txConn, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("could not begin transaction: %w", err)
	}
	if err := fn(txConn); err != nil {
		_ = txConn.Rollback(ctx)
		return err
	}
	if err := txConn.Commit(ctx); err != nil {
		_ = txConn.Rollback(ctx)
		return fmt.Errorf("could not commit transaction: %w", err)
	}
	return nil
}
