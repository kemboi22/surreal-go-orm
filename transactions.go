package surrealgoorm

import (
	"context"
	"errors"
	"sync"

	"github.com/surrealdb/surrealdb.go"
)

// ErrTransactionClosed is returned when Commit or Cancel is called on a
// transaction that has already been committed or canceled.
var ErrTransactionClosed = errors.New("surrealgoorm: transaction is already closed")

// Transaction is an interactive SurrealDB transaction on a WebSocket
// connection. It exposes the same ORM builder as Query via the Query method,
// so every operation performed inside the transaction is atomic and can be
// rolled back with Cancel.
type Transaction struct {
	tx   *surrealdb.Transaction
	mu   sync.Mutex
	done bool
}

// Begin starts a new interactive transaction on the given database connection.
//
// Interactive transactions are only supported on WebSocket connections
// (SurrealDB v3+).
//
// Example:
//
//	tx, err := surrealgoorm.Begin(ctx, db)
//	if err != nil {
//	    return err
//	}
//	defer tx.Cancel(ctx) // Cancel if not committed
//
//	user, err := tx.Query[User]("user").Create(ctx, &User{Name: "Alice"})
//	if err != nil {
//	    return err
//	}
//	_, err = tx.Query[Post]("post").Create(ctx, &Post{Title: "Hello", UserID: user.ID})
//	if err != nil {
//	    return err
//	}
//
//	return tx.Commit(ctx)
func Begin(ctx context.Context, db *surrealdb.DB) (*Transaction, error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &Transaction{tx: tx}, nil
}

// Commit commits the transaction, making all changes permanent. After Commit,
// the transaction cannot be used anymore.
func (t *Transaction) Commit(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.done {
		return ErrTransactionClosed
	}
	if err := t.tx.Commit(ctx); err != nil {
		return err
	}
	t.done = true
	return nil
}

// Cancel cancels the transaction, discarding all changes. After Cancel, the
// transaction cannot be used anymore.
func (t *Transaction) Cancel(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.done {
		return ErrTransactionClosed
	}
	if err := t.tx.Cancel(ctx); err != nil {
		return err
	}
	t.done = true
	return nil
}

// IsClosed returns whether the transaction has been committed or canceled.
func (t *Transaction) IsClosed() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.done
}

// Query returns an ORM query builder scoped to this transaction.
//
// Example:
//
//	user, err := tx.Query[User]("user").Create(ctx, &User{Name: "Alice"})
func (t *Transaction) Query[T any](table string) *Model[T] {
	return newModel[T](t.tx, table)
}

// QueryTx returns an ORM query builder scoped to the given transaction.
//
// Deprecated: use tx.Query[T](table) instead.
//
// Example:
//
//	user, err := surrealgoorm.QueryTx[User](tx, "user").Create(ctx, &User{Name: "Alice"})
func QueryTx[T any](tx *Transaction, table string) *Model[T] {
	return tx.Query[T](table)
}

// Raw runs a raw SurrealQL statement inside the transaction and returns the
// first result set.
func (t *Transaction) Raw(ctx context.Context, sql string, vars map[string]any) ([]map[string]any, error) {
	res, err := runQuery(ctx, t.tx, sql, vars)
	if err != nil {
		return nil, err
	}
	if len(*res) == 0 {
		return nil, nil
	}
	return (*res)[0].Result, nil
}
