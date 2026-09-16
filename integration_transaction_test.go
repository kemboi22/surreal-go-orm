package surrealgoorm

import (
	"context"
	"errors"
	"testing"
)

type txUser struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
}

func TestIntegrationTransactionCommit(t *testing.T) {
	skipIfNoDB(t)
	db := getTestDB(t)
	defer db.Close(context.Background())
	ctx := context.Background()

	table := "test_tx_users"
	defineEloquentSchema(t, ctx, db, table, []string{"name"})
	defer dropTable(t, ctx, db, table)

	tx, err := Begin(ctx, db)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}
	if tx.IsClosed() {
		t.Fatal("new transaction should not be closed")
	}

	created, err := tx.Query[txUser](table).Create(ctx, &txUser{Name: "Alice"})
	if err != nil {
		tx.Cancel(ctx)
		t.Fatalf("Create in tx failed: %v", err)
	}
	if created.ID == "" {
		tx.Cancel(ctx)
		t.Fatal("Create in tx should populate the id")
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("Commit failed: %v", err)
	}
	if !tx.IsClosed() {
		t.Fatal("committed transaction should be closed")
	}

	found, err := Query[txUser](db, table).Find(ctx, created.ID)
	if err != nil {
		t.Fatalf("Find after commit failed: %v", err)
	}
	if found == nil || found.Name != "Alice" {
		t.Fatalf("expected committed record, got %+v", found)
	}

	if err := tx.Commit(ctx); !errors.Is(err, ErrTransactionClosed) {
		t.Errorf("second Commit should return ErrTransactionClosed, got %v", err)
	}
}

func TestIntegrationTransactionCancel(t *testing.T) {
	skipIfNoDB(t)
	db := getTestDB(t)
	defer db.Close(context.Background())
	ctx := context.Background()

	table := "test_tx_users_cancel"
	defineEloquentSchema(t, ctx, db, table, []string{"name"})
	defer dropTable(t, ctx, db, table)

	tx, err := Begin(ctx, db)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	created, err := tx.Query[txUser](table).Create(ctx, &txUser{Name: "Bob"})
	if err != nil {
		tx.Cancel(ctx)
		t.Fatalf("Create in tx failed: %v", err)
	}

	if err := tx.Cancel(ctx); err != nil {
		t.Fatalf("Cancel failed: %v", err)
	}
	if !tx.IsClosed() {
		t.Fatal("canceled transaction should be closed")
	}

	found, err := Query[txUser](db, table).Find(ctx, created.ID)
	if err != nil {
		t.Fatalf("Find after cancel failed: %v", err)
	}
	if found != nil {
		t.Fatalf("expected canceled record to be absent, got %+v", found)
	}

	if err := tx.Cancel(ctx); !errors.Is(err, ErrTransactionClosed) {
		t.Errorf("second Cancel should return ErrTransactionClosed, got %v", err)
	}
}

func TestIntegrationTransactionRollsBackOnError(t *testing.T) {
	skipIfNoDB(t)
	db := getTestDB(t)
	defer db.Close(context.Background())
	ctx := context.Background()

	table := "test_tx_users_rollback"
	defineEloquentSchema(t, ctx, db, table, []string{"name"})
	defer dropTable(t, ctx, db, table)

	tx, err := Begin(ctx, db)
	if err != nil {
		t.Fatalf("Begin failed: %v", err)
	}

	created, err := QueryTx[txUser](tx, table).Create(ctx, &txUser{Name: "Carol"})
	if err != nil {
		tx.Cancel(ctx)
		t.Fatalf("Create in tx failed: %v", err)
	}

	// Force a failure before committing.
	if _, err := tx.Raw(ctx, "THROW 'boom';", nil); err == nil {
		tx.Cancel(ctx)
		t.Fatal("expected THROW to fail")
	}

	if err := tx.Cancel(ctx); err != nil {
		t.Fatalf("Cancel failed: %v", err)
	}

	found, err := Query[txUser](db, table).Find(ctx, created.ID)
	if err != nil {
		t.Fatalf("Find after rollback failed: %v", err)
	}
	if found != nil {
		t.Fatalf("expected rolled-back record to be absent, got %+v", found)
	}
}
