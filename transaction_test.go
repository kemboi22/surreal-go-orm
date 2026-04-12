package surrealgoorm

import (
	"context"
	"errors"
	"testing"
)

func TestCommitRollbackWithoutTx(t *testing.T) {
	db := &DB{}
	if !errors.Is(db.Commit(context.Background()), ErrNoActiveTx) {
		t.Fatal("expected ErrNoActiveTx on commit")
	}
	if !errors.Is(db.Rollback(context.Background()), ErrNoActiveTx) {
		t.Fatal("expected ErrNoActiveTx on rollback")
	}
}

func TestTransactionBeginWithoutRawDB(t *testing.T) {
	db := &DB{}
	err := db.Transaction(context.Background(), func(tx *DB) error { return nil })
	if !errors.Is(err, ErrInvalidModel) {
		t.Fatalf("expected ErrInvalidModel, got %v", err)
	}
}
