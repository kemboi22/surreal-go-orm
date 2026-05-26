package surrealgoorm

import (
	"context"
	"testing"
)

func TestSchemaCreateTable(t *testing.T) {
	schema := &Schema{Db: nil}

	var called bool
	err := schema.CreateTable(context.Background(), "users", func(tbl *Table) {
		called = true
		if tbl.Name != "users" {
			t.Error("expected table name 'users', got", tbl.Name)
		}
	})

	if err != nil {
		t.Errorf("CreateTable returned error: %v", err)
	}
	if !called {
		t.Error("CreateTable should call the callback function")
	}
}

func TestSchemaCreateTableAddsColumns(t *testing.T) {
	schema := &Schema{Db: nil}

	var captured *Table
	err := schema.CreateTable(context.Background(), "products", func(tbl *Table) {
		tbl.String("name")
		tbl.Float("price")
		tbl.Bool("available")
		captured = tbl
	})

	if err != nil {
		t.Errorf("CreateTable returned error: %v", err)
	}
	if captured == nil {
		t.Fatal("expected callback to be called")
	}
	if len(captured.Columns) != 3 {
		t.Errorf("expected 3 columns, got %d", len(captured.Columns))
	}
	if captured.Columns[0].Name != "name" {
		t.Errorf("first column should be 'name', got '%s'", captured.Columns[0].Name)
	}
}

func TestSchemaCreateTableWithModifiers(t *testing.T) {
	schema := &Schema{Db: nil}

	var captured *Table
	err := schema.CreateTable(context.Background(), "users", func(tbl *Table) {
		tbl.String("email").Unique().Nullable()
		tbl.Int("age").Default(18)
		captured = tbl
	})

	if err != nil {
		t.Errorf("CreateTable returned error: %v", err)
	}
	if len(captured.Columns) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(captured.Columns))
	}

	email := captured.Columns[0]
	if !email.IsUnique {
		t.Error("email column should be IsUnique")
	}
	if !email.Optional {
		t.Error("email column should be Optional (Nullable)")
	}

	age := captured.Columns[1]
	if age.DefaultValue != 18 {
		t.Errorf("age column DefaultValue should be 18, got %v", age.DefaultValue)
	}
}

func TestSchemaCreateTableMultipleTables(t *testing.T) {
	schema := &Schema{Db: nil}

	var tableNames []string
	schema.CreateTable(context.Background(), "users", func(tbl *Table) {
		tableNames = append(tableNames, tbl.Name)
		tbl.String("name")
	})
	schema.CreateTable(context.Background(), "posts", func(tbl *Table) {
		tableNames = append(tableNames, tbl.Name)
		tbl.String("title")
		tbl.String("body")
	})

	if len(tableNames) != 2 {
		t.Fatalf("expected 2 tables, got %d", len(tableNames))
	}
	if tableNames[0] != "users" {
		t.Errorf("first table should be 'users', got '%s'", tableNames[0])
	}
	if tableNames[1] != "posts" {
		t.Errorf("second table should be 'posts', got '%s'", tableNames[1])
	}
}

func TestSchemaCreateTableAlwaysReturnsNil(t *testing.T) {
	schema := &Schema{Db: nil}
	err := schema.CreateTable(context.Background(), "test", func(tbl *Table) {})
	if err != nil {
		t.Errorf("CreateTable should always return nil, got: %v", err)
	}
}
