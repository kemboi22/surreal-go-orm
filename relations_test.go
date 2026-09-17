package surrealgoorm

import (
	"context"
	"strings"
	"testing"

	"github.com/surrealdb/surrealdb.go/contrib/surrealql"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

func TestWhereInBuildsINClause(t *testing.T) {
	q := Query[testUser](nil, "user").WhereIn("author_id", []any{"a", "b", "c"})
	sql, vars := q.ToBuildSQL()
	if !strings.Contains(sql, "author_id IN $") {
		t.Errorf("expected IN clause in SQL, got: %s", sql)
	}
	if len(vars) != 1 {
		t.Errorf("expected 1 var, got %d", len(vars))
	}
}

func TestSetUpdateFieldNIL(t *testing.T) {
	q := surrealql.Update("user:1")
	setUpdateField(q, "email", "a@b.com")
	setUpdateField(q, "name", nil)
	sql, _ := q.Build()
	if !strings.Contains(sql, "name = NONE") {
		t.Errorf("expected name = NONE in SQL, got: %s", sql)
	}
	if !strings.Contains(sql, "email") {
		t.Errorf("expected email set clause in SQL, got: %s", sql)
	}
}

func TestSetUpdateFieldNonNil(t *testing.T) {
	q := surrealql.Update("user:1")
	setUpdateField(q, "age", 30)
	sql, _ := q.Build()
	if !strings.Contains(sql, "age = ") {
		t.Errorf("expected age assignment in SQL, got: %s", sql)
	}
}

func TestRunQueryRejectsUnknownClient(t *testing.T) {
	_, err := runQuery(nil, "not-a-client", "SELECT 1", nil)
	if err == nil {
		t.Fatal("expected error for unsupported client type")
	}
	if !strings.Contains(err.Error(), "unsupported client type") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestClientOfModel(t *testing.T) {
	q := Query[testUser](nil, "user")
	c, err := clientOf(q)
	if err != nil {
		t.Fatalf("clientOf failed: %v", err)
	}
	if c == nil {
		t.Error("expected a (typed nil) client to be returned")
	}
}

func TestQueryReturnsConcreteModel(t *testing.T) {
	var q QueryBuilder[testUser] = Query[testUser](nil, "user").WhereEq("name", "Alice")
	if q == nil {
		t.Fatal("expected non-nil QueryBuilder")
	}
	m, ok := q.(*Model[testUser])
	if !ok {
		t.Fatalf("expected *Model[testUser], got %T", q)
	}
	if m.table != "user" {
		t.Errorf("expected table user, got %s", m.table)
	}
}

func TestDeprecatedHasManyRequiresModel(t *testing.T) {
	var parent QueryBuilder[testUser] = fakeBuilder[testUser]{}
	_, err := HasMany[testUser, testUser](nil, parent, "post", "user_id")
	if err == nil {
		t.Fatal("expected error for non-Model QueryBuilder")
	}
	if !strings.Contains(err.Error(), "does not support relationships") {
		t.Errorf("unexpected error: %v", err)
	}
}

type fakeBuilder[T any] struct{}

func (fakeBuilder[T]) Select(...string) *Model[T]                { return nil }
func (fakeBuilder[T]) Where(string, string, any) *Model[T]       { return nil }
func (fakeBuilder[T]) WhereEq(string, any) *Model[T]             { return nil }
func (fakeBuilder[T]) WhereNotNull(string) *Model[T]             { return nil }
func (fakeBuilder[T]) WhereNull(string) *Model[T]                { return nil }
func (fakeBuilder[T]) WhereIn(string, []any) *Model[T]           { return nil }
func (fakeBuilder[T]) WhereContains(string, any) *Model[T]       { return nil }
func (fakeBuilder[T]) WhereContainsAny(string, ...any) *Model[T] { return nil }
func (fakeBuilder[T]) OrderBy(string, ...string) *Model[T]       { return nil }
func (fakeBuilder[T]) OrderByDesc(string) *Model[T]              { return nil }
func (fakeBuilder[T]) Limit(int) *Model[T]                       { return nil }
func (fakeBuilder[T]) WithNames(...string) *Model[T]             { return nil }
func (fakeBuilder[T]) WithTrashed() *Model[T]                    { return nil }
func (fakeBuilder[T]) OnlyTrashed() *Model[T]                    { return nil }
func (fakeBuilder[T]) ToSQL() string                             { return "" }
func (fakeBuilder[T]) ToBuildSQL() (string, map[string]any)      { return "", nil }
func (fakeBuilder[T]) First(context.Context) (*T, error)         { return nil, nil }
func (fakeBuilder[T]) Get(context.Context) (*[]T, error)         { return nil, nil }
func (fakeBuilder[T]) Create(context.Context, *T) (*T, error)    { return nil, nil }
func (fakeBuilder[T]) CreateWithID(context.Context, any, *T) (*T, error) {
	return nil, nil
}
func (fakeBuilder[T]) CreateMany(context.Context, []T) ([]T, error) { return nil, nil }
func (fakeBuilder[T]) CreateFromMap(context.Context, map[string]any) (*T, error) {
	return nil, nil
}
func (fakeBuilder[T]) Find(context.Context, any) (*T, error) { return nil, nil }
func (fakeBuilder[T]) FindOrFail(context.Context, any) (*T, error) {
	return nil, nil
}
func (fakeBuilder[T]) FirstOrCreate(context.Context, map[string]any, ...map[string]any) (*T, error) {
	return nil, nil
}
func (fakeBuilder[T]) FirstOrNew(context.Context, map[string]any, ...map[string]any) (*T, error) {
	return nil, nil
}
func (fakeBuilder[T]) UpdateOrCreate(context.Context, map[string]any, map[string]any) (*T, error) {
	return nil, nil
}
func (fakeBuilder[T]) Save(context.Context, *T) (*T, error) { return nil, nil }
func (fakeBuilder[T]) Update(context.Context, any, map[string]any) (*T, error) {
	return nil, nil
}
func (fakeBuilder[T]) UpdateWhere(context.Context, map[string]any) ([]T, error) {
	return nil, nil
}
func (fakeBuilder[T]) Increment(context.Context, any, string, int64) (*T, error) {
	return nil, nil
}
func (fakeBuilder[T]) Decrement(context.Context, any, string, int64) (*T, error) {
	return nil, nil
}
func (fakeBuilder[T]) WhereIncrement(context.Context, string, int64) ([]T, error) {
	return nil, nil
}
func (fakeBuilder[T]) WhereDecrement(context.Context, string, int64) ([]T, error) {
	return nil, nil
}
func (fakeBuilder[T]) Delete(context.Context, any) error            { return nil }
func (fakeBuilder[T]) DeleteWhere(context.Context) error            { return nil }
func (fakeBuilder[T]) ForceDelete(context.Context, any) error       { return nil }
func (fakeBuilder[T]) Restore(context.Context, any) error           { return nil }
func (fakeBuilder[T]) Truncate(context.Context) error               { return nil }
func (fakeBuilder[T]) Count(context.Context) (int, error)           { return 0, nil }
func (fakeBuilder[T]) Exists(context.Context) (bool, error)         { return false, nil }
func (fakeBuilder[T]) Sum(context.Context, string) (float64, error) { return 0, nil }
func (fakeBuilder[T]) Avg(context.Context, string) (float64, error) { return 0, nil }
func (fakeBuilder[T]) Min(context.Context, string) (float64, error) { return 0, nil }
func (fakeBuilder[T]) Max(context.Context, string) (float64, error) { return 0, nil }
func (fakeBuilder[T]) Paginate(context.Context, int, int) (*Pagination[T], error) {
	return nil, nil
}
func (fakeBuilder[T]) Relate(context.Context, models.RecordID, string, models.RecordID, map[string]any) (map[string]any, error) {
	return nil, nil
}
