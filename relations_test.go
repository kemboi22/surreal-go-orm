package surrealgoorm

import (
	"context"
	"strings"
	"testing"

	"github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

type relationUser struct {
	Model
	Name  string         `orm:"column:name"`
	Posts []relationPost `orm:"has_many:posts;foreign_key:author_id"`
}

func (relationUser) TableName() string { return "users" }

type relationPost struct {
	Model
	Title    string        `orm:"column:title"`
	AuthorID string        `orm:"column:author_id"`
	Author   *relationUser `orm:"belongs_to:users;foreign_key:AuthorID"`
}

func (relationPost) TableName() string { return "posts" }

func TestWithNestedRelations(t *testing.T) {
	db := &DB{
		queryOverride: func(ctx context.Context, sql string, params map[string]any) (*[]surrealdb.QueryResult[[]map[string]any], error) {
			switch {
			case strings.Contains(sql, "FROM posts"):
				resp := []surrealdb.QueryResult[[]map[string]any]{{Result: []map[string]any{{
					"id":        "posts:first",
					"title":     "Hello",
					"author_id": "users:john",
				}}}}
				return &resp, nil
			default:
				resp := []surrealdb.QueryResult[[]map[string]any]{{Result: []map[string]any{}}}
				return &resp, nil
			}
		},
		selectOverride: func(ctx context.Context, what any) (any, error) {
			return map[string]any{
				"id":   "users:john",
				"name": "John",
			}, nil
		},
	}

	user := &relationUser{Model: Model{ID: mustRID("users", "john")}}
	if err := db.With(context.Background(), user, "posts.author"); err != nil {
		t.Fatalf("with failed: %v", err)
	}
	if len(user.Posts) != 1 || user.Posts[0].Author == nil || user.Posts[0].Author.Name != "John" {
		t.Fatalf("nested relation was not loaded: %+v", user)
	}
}

func mustRID(table string, id string) models.RecordID {
	return models.NewRecordID(table, id)
}
