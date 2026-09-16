package surrealgoorm

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

type eloquentUser struct {
	ID        string     `json:"id,omitempty"`
	Name      string     `json:"name"`
	Email     string     `json:"email"`
	Age       int        `json:"age"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at"`
}

type eloquentPost struct {
	ID    string `json:"id,omitempty"`
	Title string `json:"title"`
}

func defineEloquentSchema(t *testing.T, ctx context.Context, db *surrealdb.DB, table string, fields []string) {
	t.Helper()
	surrealdb.Query[any](ctx, db, "DEFINE TABLE "+table+" SCHEMAFULL;", nil)
	for _, f := range fields {
		surrealdb.Query[any](ctx, db, "DEFINE FIELD "+f+" ON "+table+";", nil)
	}
}

func dropTable(t *testing.T, ctx context.Context, db *surrealdb.DB, table string) {
	t.Helper()
	surrealdb.Query[any](ctx, db, "REMOVE TABLE "+table+";", nil)
}

func TestIntegrationCreateAndFind(t *testing.T) {
	skipIfNoDB(t)
	db := getTestDB(t)
	defer db.Close(context.Background())
	ctx := context.Background()

	table := "test_eloquent_users"
	defineEloquentSchema(t, ctx, db, table, []string{"name", "email", "age", "created_at", "updated_at", "deleted_at"})
	defer dropTable(t, ctx, db, table)

	u := &eloquentUser{Name: "Alice", Email: "alice@example.com", Age: 30}
	created, err := Query[eloquentUser](db, table).Create(ctx, u)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if created.ID == "" {
		t.Error("Create should populate the record id")
	}
	if created.CreatedAt.IsZero() || created.UpdatedAt.IsZero() {
		t.Error("Create should populate timestamps")
	}

	found, err := Query[eloquentUser](db, table).Find(ctx, created.ID)
	if err != nil {
		t.Fatalf("Find failed: %v", err)
	}
	if found == nil {
		t.Fatal("Find should return the record")
	}
	if found.Name != "Alice" || found.Email != "alice@example.com" || found.Age != 30 {
		t.Errorf("unexpected record: %+v", found)
	}

	_, err = Query[eloquentUser](db, table).FindOrFail(ctx, "nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("FindOrFail should return ErrNotFound, got %v", err)
	}
}

func TestIntegrationUpdateAndSave(t *testing.T) {
	skipIfNoDB(t)
	db := getTestDB(t)
	defer db.Close(context.Background())
	ctx := context.Background()

	table := "test_eloquent_updates"
	defineEloquentSchema(t, ctx, db, table, []string{"name", "email", "age", "created_at", "updated_at", "deleted_at"})
	defer dropTable(t, ctx, db, table)

	u := &eloquentUser{Name: "Bob", Email: "bob@example.com"}
	created, err := Query[eloquentUser](db, table).Create(ctx, u)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	updated, err := Query[eloquentUser](db, table).Update(ctx, created.ID, map[string]any{"age": 40})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if updated.Age != 40 {
		t.Errorf("expected age 40, got %d", updated.Age)
	}

	created.Name = "Robert"
	saved, err := Query[eloquentUser](db, table).Save(ctx, created)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if saved.Name != "Robert" {
		t.Errorf("expected saved name Robert, got %s", saved.Name)
	}
}

func TestIntegrationSoftDelete(t *testing.T) {
	skipIfNoDB(t)
	db := getTestDB(t)
	defer db.Close(context.Background())
	ctx := context.Background()

	table := "test_eloquent_softdelete"
	defineEloquentSchema(t, ctx, db, table, []string{"name", "email", "age", "created_at", "updated_at", "deleted_at"})
	defer dropTable(t, ctx, db, table)

	u := &eloquentUser{Name: "Carol", Email: "carol@example.com"}
	created, err := Query[eloquentUser](db, table).Create(ctx, u)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := Query[eloquentUser](db, table).Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	missing, err := Query[eloquentUser](db, table).Find(ctx, created.ID)
	if err != nil {
		t.Fatalf("Find after delete failed: %v", err)
	}
	if missing != nil {
		t.Error("default scope should exclude soft-deleted records")
	}

	trashed, err := Query[eloquentUser](db, table).OnlyTrashed().Find(ctx, created.ID)
	if err != nil {
		t.Fatalf("OnlyTrashed Find failed: %v", err)
	}
	if trashed == nil || trashed.DeletedAt == nil {
		t.Error("OnlyTrashed should find the soft-deleted record with deleted_at set")
	}

	if err := Query[eloquentUser](db, table).Restore(ctx, created.ID); err != nil {
		t.Fatalf("Restore failed: %v", err)
	}
	back, err := Query[eloquentUser](db, table).Find(ctx, created.ID)
	if err != nil {
		t.Fatalf("Find after restore failed: %v", err)
	}
	if back == nil {
		t.Error("record should be visible again after restore")
	}

	if err := Query[eloquentUser](db, table).ForceDelete(ctx, created.ID); err != nil {
		t.Fatalf("ForceDelete failed: %v", err)
	}
	gone, err := Query[eloquentUser](db, table).WithTrashed().Find(ctx, created.ID)
	if err != nil {
		t.Fatalf("Find after force delete failed: %v", err)
	}
	if gone != nil {
		t.Error("ForceDelete should remove the record permanently")
	}
}

func TestIntegrationFirstOrCreateAndAggregates(t *testing.T) {
	skipIfNoDB(t)
	db := getTestDB(t)
	defer db.Close(context.Background())
	ctx := context.Background()

	table := "test_eloquent_findcreate"
	defineEloquentSchema(t, ctx, db, table, []string{"name", "email", "age", "created_at", "updated_at", "deleted_at"})
	defer dropTable(t, ctx, db, table)

	first, err := Query[eloquentUser](db, table).FirstOrCreate(ctx, map[string]any{"email": "dave@example.com"}, map[string]any{"name": "Dave", "age": 25})
	if err != nil {
		t.Fatalf("FirstOrCreate failed: %v", err)
	}
	if first.Name != "Dave" {
		t.Errorf("expected created user Dave, got %+v", first)
	}

	again, err := Query[eloquentUser](db, table).FirstOrCreate(ctx, map[string]any{"email": "dave@example.com"}, map[string]any{"name": "ShouldNotCreate"})
	if err != nil {
		t.Fatalf("Second FirstOrCreate failed: %v", err)
	}
	if again.Name != "Dave" {
		t.Errorf("Second FirstOrCreate should find existing, got %+v", again)
	}

	count, err := Query[eloquentUser](db, table).Count(ctx)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}

	exists, err := Query[eloquentUser](db, table).WhereEq("email", "dave@example.com").Exists(ctx)
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("Exists should be true")
	}

	if err := Query[eloquentUser](db, table).Delete(ctx, first.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	countAfterDelete, err := Query[eloquentUser](db, table).Count(ctx)
	if err != nil {
		t.Fatalf("Count after delete failed: %v", err)
	}
	if countAfterDelete != 0 {
		t.Errorf("soft-deleted records should be excluded from count, got %d", countAfterDelete)
	}
}

func TestIntegrationCreateManyAndPaginate(t *testing.T) {
	skipIfNoDB(t)
	db := getTestDB(t)
	defer db.Close(context.Background())
	ctx := context.Background()

	table := "test_eloquent_bulk"
	defineEloquentSchema(t, ctx, db, table, []string{"name", "email", "age", "created_at", "updated_at", "deleted_at"})
	defer dropTable(t, ctx, db, table)

	items := []eloquentUser{
		{Name: "One", Age: 1},
		{Name: "Two", Age: 2},
		{Name: "Three", Age: 3},
		{Name: "Four", Age: 4},
		{Name: "Five", Age: 5},
	}
	created, err := Query[eloquentUser](db, table).CreateMany(ctx, items)
	if err != nil {
		t.Fatalf("CreateMany failed: %v", err)
	}
	if len(created) != 5 {
		t.Fatalf("expected 5 created records, got %d", len(created))
	}
	for _, c := range created {
		if c.ID == "" {
			t.Error("CreateMany should populate ids")
		}
	}

	page, err := Query[eloquentUser](db, table).OrderBy("age").Paginate(ctx, 2, 2)
	if err != nil {
		t.Fatalf("Paginate failed: %v", err)
	}
	if page.Total != 5 {
		t.Errorf("expected total 5, got %d", page.Total)
	}
	if page.PerPage != 2 || page.CurrentPage != 2 {
		t.Errorf("unexpected page metadata: %+v", page)
	}
	if page.LastPage != 3 {
		t.Errorf("expected last page 3, got %d", page.LastPage)
	}
	if len(page.Data) != 2 {
		t.Fatalf("expected 2 items on page 2, got %d", len(page.Data))
	}
	if page.Data[0].Name != "Three" || page.Data[1].Name != "Four" {
		t.Errorf("unexpected page data order: %+v", page.Data)
	}
}

func TestIntegrationRelate(t *testing.T) {
	skipIfNoDB(t)
	db := getTestDB(t)
	defer db.Close(context.Background())
	ctx := context.Background()

	table := "test_eloquent_relate"
	defineEloquentSchema(t, ctx, db, table, []string{"title"})
	edge := "test_eloquent_likes"
	surrealdb.Query[any](ctx, db, "DEFINE TABLE "+edge+" TYPE RELATION FROM "+table+" TO "+table+";", nil)
	defer dropTable(t, ctx, db, table)
	defer dropTable(t, ctx, db, edge)

	post := &eloquentPost{Title: "hello"}
	created, err := Query[eloquentPost](db, table).Create(ctx, post)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	from := models.NewRecordID(table, "alice")
	to := models.NewRecordID(table, created.ID)
	rel, err := Query[eloquentPost](db, table).Relate(ctx, from, edge, to, map[string]any{"kind": "like"})
	if err != nil {
		t.Fatalf("Relate failed: %v", err)
	}
	if len(rel) == 0 {
		t.Error("Relate should return the created edge")
	}
}
