package surrealgoorm

import (
	"context"
	"errors"
	"testing"
)

type softPost struct {
	Model
	Title string `orm:"column:title"`
	SoftDeletesModel
}

func (softPost) TableName() string {
	return "posts"
}

func TestModelQuerySoftDeleteSQL(t *testing.T) {
	sql := QueryModel[softPost](context.Background(), &DB{}).buildSQL()
	if want := "SELECT * FROM posts WHERE deleted_at IS NONE"; sql != want {
		t.Fatalf("unexpected sql\nwant: %s\ngot:  %s", want, sql)
	}
}

func TestQueryBuilderSoftDeleteSQL(t *testing.T) {
	sql := (&DB{}).Query("posts").SoftDeletes().OnlyTrashed().buildSQL()
	if want := "SELECT * FROM posts WHERE deleted_at IS NOT NONE"; sql != want {
		t.Fatalf("unexpected sql\nwant: %s\ngot:  %s", want, sql)
	}
}

func TestQueryBuilderMutationGuard(t *testing.T) {
	err := (&DB{}).Query("posts").Update(context.Background(), map[string]any{"title": "x"})
	if !errors.Is(err, ErrUnsafeMutation) {
		t.Fatalf("expected ErrUnsafeMutation, got %v", err)
	}
}

func TestTableQueryMutationGuard(t *testing.T) {
	err := (&DB{}).Table("posts").Delete(context.Background())
	if !errors.Is(err, ErrUnsafeMutation) {
		t.Fatalf("expected ErrUnsafeMutation, got %v", err)
	}
}

func TestQueryBuilderWhereFeatures(t *testing.T) {
	sql, params := (&DB{}).Query("posts").
		WhereLike("title", "go").
		WhereExists("SELECT id FROM comments WHERE comments.post_id = posts.id").
		OrderByRaw("rand()").
		SelectRaw("count() as total").
		SQL()

	want := "SELECT count() as total FROM posts WHERE title CONTAINS $w0 AND (EXISTS(SELECT id FROM comments WHERE comments.post_id = posts.id)) ORDER BY rand()"
	if sql != want {
		t.Fatalf("unexpected sql\nwant: %s\ngot:  %s", want, sql)
	}
	if params["w0"] != "go" {
		t.Fatalf("expected like binding, got %#v", params)
	}
}
