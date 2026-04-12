package surrealgoorm

import (
	"context"
	"testing"

	"github.com/surrealdb/surrealdb.go/pkg/models"
)

type hookUser struct {
	Model
	Name   string   `orm:"column:name"`
	Events []string `orm:"-"`
}

func (hookUser) TableName() string { return "users" }

func (u *hookUser) BeforeCreate(ctx context.Context, db any) error {
	u.Events = append(u.Events, "before_create")
	return nil
}

func (u *hookUser) AfterCreate(ctx context.Context, db any) error {
	u.Events = append(u.Events, "after_create")
	return nil
}

func (u *hookUser) BeforeSave(ctx context.Context, db any) error {
	u.Events = append(u.Events, "before_save")
	return nil
}

func (u *hookUser) AfterSave(ctx context.Context, db any) error {
	u.Events = append(u.Events, "after_save")
	return nil
}

func (u *hookUser) BeforeUpdate(ctx context.Context, db any) error {
	u.Events = append(u.Events, "before_update")
	return nil
}

func (u *hookUser) AfterUpdate(ctx context.Context, db any) error {
	u.Events = append(u.Events, "after_update")
	return nil
}

func (u *hookUser) BeforeDelete(ctx context.Context, db any) error {
	u.Events = append(u.Events, "before_delete")
	return nil
}

func (u *hookUser) AfterDelete(ctx context.Context, db any) error {
	u.Events = append(u.Events, "after_delete")
	return nil
}

func TestCreateUpdateDeleteHooks(t *testing.T) {
	db := &DB{
		createOverride: func(ctx context.Context, what any, data any) (any, error) { return map[string]any{}, nil },
		updateOverride: func(ctx context.Context, what any, data any) (any, error) { return map[string]any{}, nil },
		deleteOverride: func(ctx context.Context, what any) (any, error) { return map[string]any{}, nil },
		selectOverride: func(ctx context.Context, what any) (any, error) { return nil, nil },
	}

	user := &hookUser{Name: "John"}
	if err := Create(context.Background(), db, user); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	user.ID = models.NewRecordID("users", "john")
	if err := Update(context.Background(), db, user); err != nil {
		t.Fatalf("update failed: %v", err)
	}
	if err := Delete(context.Background(), db, user); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	want := []string{
		"before_save", "before_create", "after_create", "after_save",
		"before_save", "before_update", "after_update", "after_save",
		"before_delete", "after_delete",
	}
	if len(user.Events) != len(want) {
		t.Fatalf("unexpected events: %#v", user.Events)
	}
	for i, event := range want {
		if user.Events[i] != event {
			t.Fatalf("event %d mismatch: want %s got %s", i, event, user.Events[i])
		}
	}
}
