package surrealgoorm

import (
	"context"
	"os"
	"testing"

	"github.com/surrealdb/surrealdb.go"
)

func getTestDB(t *testing.T) *surrealdb.DB {
	t.Helper()
	ctx := context.Background()
	url := os.Getenv("SURREALDB_URL")
	if url == "" {
		url = "ws://localhost:8000/rpc"
	}
	user := os.Getenv("SURREALDB_USER")
	if user == "" {
		user = "root"
	}
	pass := os.Getenv("SURREALDB_PASS")
	if pass == "" {
		pass = "root"
	}
	ns := os.Getenv("SURREALDB_NS")
	if ns == "" {
		ns = "test"
	}
	dbName := os.Getenv("SURREALDB_DB")
	if dbName == "" {
		dbName = "test"
	}

	db, err := surrealdb.FromEndpointURLString(ctx, url)
	if err != nil {
		t.Fatalf("Failed to connect to SurrealDB at %s: %v", url, err)
	}

	if _, err := db.SignIn(ctx, surrealdb.Auth{
		Username: user,
		Password: pass,
	}); err != nil {
		t.Fatalf("Failed to sign in: %v", err)
	}

	if err := db.Use(ctx, ns, dbName); err != nil {
		t.Fatalf("Failed to use namespace/db: %v", err)
	}

	return db
}

func skipIfNoDB(t *testing.T) {
	t.Helper()
	if os.Getenv("SURREALDB_INTEGRATION") == "" {
		t.Skip("Skipping integration test. Set SURREALDB_INTEGRATION=1 to enable.")
	}
}

func TestIntegrationCreateTable(t *testing.T) {
	skipIfNoDB(t)
	db := getTestDB(t)
	defer db.Close(context.Background())

	ctx := context.Background()
	schema := &Schema{Db: db}

	tableName := "test_integration_users"
	surrealdb.Query[any](ctx, db, "DEFINE TABLE "+tableName+" SCHEMAFULL;", nil)

	var captured *Table
	err := schema.CreateTable(ctx, tableName, func(t *Table) {
		t.String("full_name")
		t.Int("age").Default(0)
		t.Bool("is_active").Default(true)
		captured = t
	})
	if err != nil {
		t.Fatalf("CreateTable failed: %v", err)
	}
	if captured == nil {
		t.Fatal("callback was not invoked")
	}

	built := captured.Build()
	if built == "" {
		t.Fatal("Build() returned empty SQL")
	}

	surrealdb.Query[any](ctx, db, "REMOVE TABLE "+tableName+";", nil)
}

func TestIntegrationQueryBuilder(t *testing.T) {
	skipIfNoDB(t)
	db := getTestDB(t)
	defer db.Close(context.Background())

	ctx := context.Background()

	tableName := "test_integration_items"
	surrealdb.Query[any](ctx, db, "DEFINE TABLE "+tableName+" SCHEMAFULL;", nil)
	surrealdb.Query[any](ctx, db, "DEFINE FIELD name ON "+tableName+" type string;", nil)
	surrealdb.Query[any](ctx, db, "DEFINE FIELD price ON "+tableName+" type int;", nil)

	type Item struct {
		ID    string `json:"id,omitempty"`
		Name  string `json:"name"`
		Price int    `json:"price"`
	}

	surrealdb.Query[any](ctx, db, "CREATE "+tableName+" CONTENT { name: 'apple', price: 100 };", nil)
	surrealdb.Query[any](ctx, db, "CREATE "+tableName+" CONTENT { name: 'banana', price: 50 };", nil)

	qb := Query[Item](db, tableName)
	results, err := qb.Get(ctx)
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}
	if results == nil {
		t.Fatal("Get() returned nil results")
	}

	surrealdb.Query[any](ctx, db, "REMOVE TABLE "+tableName+";", nil)
}

func TestIntegrationQueryWithWhere(t *testing.T) {
	skipIfNoDB(t)
	db := getTestDB(t)
	defer db.Close(context.Background())

	ctx := context.Background()

	tableName := "test_integration_where"
	surrealdb.Query[any](ctx, db, "DEFINE TABLE "+tableName+" SCHEMAFULL;", nil)
	surrealdb.Query[any](ctx, db, "DEFINE FIELD name ON "+tableName+" type string;", nil)
	surrealdb.Query[any](ctx, db, "DEFINE FIELD score ON "+tableName+" type int;", nil)

	type Record struct {
		ID    string `json:"id,omitempty"`
		Name  string `json:"name"`
		Score int    `json:"score"`
	}

	surrealdb.Query[any](ctx, db, "CREATE "+tableName+" CONTENT { name: 'alice', score: 95 };", nil)
	surrealdb.Query[any](ctx, db, "CREATE "+tableName+" CONTENT { name: 'bob', score: 80 };", nil)
	surrealdb.Query[any](ctx, db, "CREATE "+tableName+" CONTENT { name: 'charlie', score: 70 };", nil)

	result, err := Query[Record](db, tableName).
		WhereEq("name", "bob").
		First(ctx)

	if err != nil {
		t.Fatalf("First() with WhereEq failed: %v", err)
	}
	if result == nil {
		t.Fatal("First() should find bob")
	}
	if result.Name != "bob" {
		t.Errorf("expected bob, got %s", result.Name)
	}

	surrealdb.Query[any](ctx, db, "REMOVE TABLE "+tableName+";", nil)
}

func TestIntegrationMigration(t *testing.T) {
	skipIfNoDB(t)
	db := getTestDB(t)
	defer db.Close(context.Background())

	ctx := context.Background()

	tableName := "test_integration_migration"

	m := &integrationMigration{name: "test_migrate", tableName: tableName, db: db}

	schema := &Schema{Db: db}
	err := m.Up(ctx, *schema)
	if err != nil {
		t.Fatalf("Migration Up() failed: %v", err)
	}

	err = m.Down(ctx, *schema)
	if err != nil {
		t.Fatalf("Migration Down() failed: %v", err)
	}
}

type integrationMigration struct {
	name      string
	tableName string
	db        *surrealdb.DB
}

func (m *integrationMigration) Name() string {
	return m.name
}

func (m *integrationMigration) Up(ctx context.Context, schema Schema) error {
	schema.CreateTable(ctx, m.tableName, func(t *Table) {
		t.String("data")
	})
	built := (&Table{Name: m.tableName}).Build()
	_, err := surrealdb.Query[any](ctx, m.db, built, nil)
	return err
}

func (m *integrationMigration) Down(ctx context.Context, schema Schema) error {
	_, err := surrealdb.Query[any](ctx, m.db, "REMOVE TABLE "+m.tableName+";", nil)
	return err
}
