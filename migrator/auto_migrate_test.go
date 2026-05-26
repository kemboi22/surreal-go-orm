package migrator

import (
	"context"
	"testing"

	surrealgoorm "github.com/kemboi22/surreal-go-orm"
)

type mockMigration struct {
	name       string
	upCalled   bool
	downCalled bool
	upErr      error
	downErr    error
}

func (m *mockMigration) Name() string {
	return m.name
}

func (m *mockMigration) Up(ctx context.Context, schema surrealgoorm.Schema) error {
	m.upCalled = true
	return m.upErr
}

func (m *mockMigration) Down(ctx context.Context, schema surrealgoorm.Schema) error {
	m.downCalled = true
	return m.downErr
}

func TestNewAutoMigrator(t *testing.T) {
	m := NewAutoMigrator(nil)
	if m == nil {
		t.Fatal("NewAutoMigrator() should not return nil")
	}
	if m.Db != nil {
		t.Error("NewAutoMigrator(nil) should have nil Db")
	}
}

func TestAutoMigrateNoModels(t *testing.T) {
	m := NewAutoMigrator(nil)
	err := m.AutoMigrate(context.Background(), []surrealgoorm.Migration{})
	if err != nil {
		t.Errorf("AutoMigrate with no models should succeed, got: %v", err)
	}
}

func TestAutoMigrateSingleModel(t *testing.T) {
	m := NewAutoMigrator(nil)
	mm := &mockMigration{name: "test"}
	err := m.AutoMigrate(context.Background(), []surrealgoorm.Migration{mm})
	if err != nil {
		t.Errorf("AutoMigrate should succeed, got: %v", err)
	}
	if !mm.upCalled {
		t.Error("AutoMigrate should call Up() on the migration")
	}
}

func TestAutoMigrateMultipleModels(t *testing.T) {
	m := NewAutoMigrator(nil)
	m1 := &mockMigration{name: "001_users"}
	m2 := &mockMigration{name: "002_posts"}
	m3 := &mockMigration{name: "003_comments"}

	err := m.AutoMigrate(context.Background(), []surrealgoorm.Migration{m1, m2, m3})
	if err != nil {
		t.Errorf("AutoMigrate should succeed, got: %v", err)
	}
	if !m1.upCalled || !m2.upCalled || !m3.upCalled {
		t.Error("AutoMigrate should call Up() on all migrations")
	}
}

func TestAutoMigrateStopsOnError(t *testing.T) {
	m := NewAutoMigrator(nil)
	m1 := &mockMigration{name: "001", upErr: assertAnError}
	m2 := &mockMigration{name: "002"}

	err := m.AutoMigrate(context.Background(), []surrealgoorm.Migration{m1, m2})
	if err == nil {
		t.Error("AutoMigrate should return error when a migration fails")
	}
	if m2.upCalled {
		t.Error("AutoMigrate should stop after first error, but m2 was called")
	}
}

type customError struct{}

func (e customError) Error() string {
	return "custom error"
}

var assertAnError customError

func TestAutoMigratePreservesOrder(t *testing.T) {
	m := NewAutoMigrator(nil)
	var callOrder []string
	m1 := &orderedMock{name: "first", order: &callOrder}
	m2 := &orderedMock{name: "second", order: &callOrder}

	m.AutoMigrate(context.Background(), []surrealgoorm.Migration{m1, m2})
	if len(callOrder) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(callOrder))
	}
	if callOrder[0] != "first" || callOrder[1] != "second" {
		t.Errorf("migrations called out of order: %v", callOrder)
	}
}

type orderedMock struct {
	name  string
	order *[]string
	upErr error
}

func (m *orderedMock) Name() string {
	return m.name
}

func (m *orderedMock) Up(ctx context.Context, schema surrealgoorm.Schema) error {
	*m.order = append(*m.order, m.name)
	return m.upErr
}

func (m *orderedMock) Down(ctx context.Context, schema surrealgoorm.Schema) error {
	return nil
}
