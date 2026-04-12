package surrealgoorm

import (
	"context"
	"testing"

	"github.com/surrealdb/surrealdb.go"
)

func TestModelQueryWithTrashedSQL(t *testing.T) {
	sql := QueryModel[softPost](context.Background(), &DB{}).WithTrashed().buildSQL()
	if want := "SELECT * FROM posts"; sql != want {
		t.Fatalf("unexpected sql\nwant: %s\ngot:  %s", want, sql)
	}
}

func TestQueryBuilderDeleteUsesSoftDeleteWhenEnabled(t *testing.T) {
	var gotSQL string
	db := &DB{
		execOverride: func(ctx context.Context, sql string, params map[string]any) (*[]surrealdb.QueryResult[any], error) {
			gotSQL = sql
			resp := []surrealdb.QueryResult[any]{{}}
			return &resp, nil
		},
	}

	err := db.Query("posts").SoftDeletes().Where("id", "=", "posts:first").Delete(context.Background())
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if want := "UPDATE posts SET deleted_at = time::now() WHERE string::concat(id) = $w0"; gotSQL != want {
		t.Fatalf("unexpected sql\nwant: %s\ngot:  %s", want, gotSQL)
	}
}
