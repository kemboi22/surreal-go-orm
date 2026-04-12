package surrealgoorm

import (
	"testing"
	"time"

	"github.com/surrealdb/surrealdb.go/pkg/models"
)

type mapUser struct {
	Model
	Name string `orm:"column:name"`
	Age  int    `orm:"column:age"`
}

func (mapUser) TableName() string {
	return "users"
}

func TestMapToStruct(t *testing.T) {
	row := map[string]any{
		"id":   "users:john",
		"name": "John",
		"age":  float64(30),
	}

	var user mapUser
	if err := mapToStruct(row, &user); err != nil {
		t.Fatalf("mapToStruct returned error: %v", err)
	}
	if user.ID.String() != "users:john" || user.Name != "John" || user.Age != 30 {
		t.Fatalf("unexpected mapped user: %+v", user)
	}
}

func TestStructToMap(t *testing.T) {
	now := time.Now()
	user := mapUser{
		Model: Model{
			ID:        models.NewRecordID("users", "john"),
			CreatedAt: now,
			UpdatedAt: now,
		},
		Name: "John",
		Age:  30,
	}

	result, err := structToMap(user)
	if err != nil {
		t.Fatalf("structToMap returned error: %v", err)
	}
	if _, ok := result["created_at"]; ok {
		t.Fatalf("auto field should not be in result: %#v", result)
	}
	rid := result["id"].(models.RecordID)
	if (&rid).String() != "users:john" {
		t.Fatalf("missing id in result: %#v", result)
	}
}
