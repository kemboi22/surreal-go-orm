package surrealgoorm

import (
	"context"
	"testing"
)

type mockMigration struct {
	name         string
	upCalled     bool
	downCalled   bool
	upShouldErr  bool
	downShouldErr bool
}

func (m *mockMigration) Name() string {
	return m.name
}

func (m *mockMigration) Up(ctx context.Context, schema Schema) error {
	m.upCalled = true
	if m.upShouldErr {
		return assertAnError
	}
	return nil
}

func (m *mockMigration) Down(ctx context.Context, schema Schema) error {
	m.downCalled = true
	if m.downShouldErr {
		return assertAnError
	}
	return nil
}

type testError struct{}

func (e testError) Error() string {
	return "test error"
}

var assertAnError testError

func TestMigrationInterface(t *testing.T) {
	var m Migration = &mockMigration{name: "test_migration"}
	if m.Name() != "test_migration" {
		t.Errorf("expected name 'test_migration', got '%s'", m.Name())
	}
}

func TestMigrationUp(t *testing.T) {
	m := &mockMigration{name: "test"}
	schema := Schema{Db: nil}
	err := m.Up(context.Background(), schema)
	if err != nil {
		t.Errorf("Up() returned unexpected error: %v", err)
	}
	if !m.upCalled {
		t.Error("Up() was not called")
	}
}

func TestMigrationDown(t *testing.T) {
	m := &mockMigration{name: "test"}
	schema := Schema{Db: nil}
	err := m.Down(context.Background(), schema)
	if err != nil {
		t.Errorf("Down() returned unexpected error: %v", err)
	}
	if !m.downCalled {
		t.Error("Down() was not called")
	}
}

func TestMigrationUpError(t *testing.T) {
	m := &mockMigration{name: "test", upShouldErr: true}
	schema := Schema{Db: nil}
	err := m.Up(context.Background(), schema)
	if err == nil {
		t.Error("Up() should return error when upShouldErr is true")
	}
}

func TestMigrationDownError(t *testing.T) {
	m := &mockMigration{name: "test", downShouldErr: true}
	schema := Schema{Db: nil}
	err := m.Down(context.Background(), schema)
	if err == nil {
		t.Error("Down() should return error when downShouldErr is true")
	}
}

func TestMultipleMigrations(t *testing.T) {
	m1 := &mockMigration{name: "001_create_users"}
	m2 := &mockMigration{name: "002_create_posts"}
	migrations := []Migration{m1, m2}

	schema := Schema{Db: nil}
	for _, m := range migrations {
		if err := m.Up(context.Background(), schema); err != nil {
			t.Errorf("migration %s failed: %v", m.Name(), err)
		}
	}

	if !m1.upCalled {
		t.Error("first migration Up() was not called")
	}
	if !m2.upCalled {
		t.Error("second migration Up() was not called")
	}
}
