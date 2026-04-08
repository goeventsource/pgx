// Package pgx provides PostgreSQL-based implementations of the following interfaces:
// goeventsource.Repository, goeventsource.Snapshotter, and goeventsource.Store interfaces.
//
// These implementations seamlessly integrate with PostgreSQL databases,
// providing robust functionality for various data operations.
//
// Use [InTransaction] with a context from [WithValueTx] when you need several operations
// (for example snapshot read plus event stream) to share one transaction. [Store.Stream] and
// [Snapshotter.ReadSnapshot] use the transaction from the context when present.
package pgx
