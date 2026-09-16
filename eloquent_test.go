package surrealgoorm

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/surrealdb/surrealdb.go/pkg/models"
)

type testUser struct {
	ID        string     `json:"id,omitempty"`
	Name      string     `json:"name"`
	Email     string     `json:"email" orm:"fillable"`
	Password  string     `json:"password" orm:"guarded"`
	Age       int        `json:"age"`
	Active    bool       `json:"active"`
	Score     float64    `json:"score"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at"`
}

func TestReflectMetaDetects(t *testing.T) {
	meta := reflectMeta(reflect.TypeOf(testUser{}))
	if !meta.timestamps {
		t.Error("meta should detect timestamps")
	}
	if !meta.softDelete {
		t.Error("meta should detect soft delete")
	}
	if !meta.hasID {
		t.Error("meta should detect id")
	}
	if !meta.fillable["email"] {
		t.Error("meta should mark email as fillable")
	}
	if !meta.guarded["password"] {
		t.Error("meta should mark password as guarded")
	}
}

func TestStructToMap(t *testing.T) {
	m := map[string]any{
		"name":  "Alice",
		"email": "alice@example.com",
	}
	got, err := structToMap(m)
	if err == nil {
		t.Fatal("structToMap should reject maps")
	}
	_ = got

	u := testUser{Name: "Alice", Email: "alice@example.com", Age: 30}
	attrs, err := structToMap(&u)
	if err != nil {
		t.Fatalf("structToMap failed: %v", err)
	}
	if attrs["name"] != "Alice" {
		t.Errorf("expected name Alice, got %v", attrs["name"])
	}
	if attrs["email"] != "alice@example.com" {
		t.Errorf("expected email, got %v", attrs["email"])
	}
	if attrs["age"] != 30 {
		t.Errorf("expected age 30, got %v", attrs["age"])
	}
	if _, ok := attrs["id"]; !ok {
		t.Error("id should be present in the map")
	}
	if attrs["id"] != "" {
		t.Errorf("expected empty id, got %v", attrs["id"])
	}
}

func TestMapToStructRecordID(t *testing.T) {
	raw := map[string]any{
		"id":    models.RecordID{Table: "users", ID: "abc123"},
		"name":  "Alice",
		"email": "alice@example.com",
		"age":   30,
		"score": 9.5,
	}
	var u testUser
	if err := mapToStruct(raw, &u); err != nil {
		t.Fatalf("mapToStruct failed: %v", err)
	}
	if u.ID != "users:abc123" {
		t.Errorf("expected id users:abc123, got %q", u.ID)
	}
	if u.Name != "Alice" {
		t.Errorf("expected name Alice, got %q", u.Name)
	}
	if u.Age != 30 {
		t.Errorf("expected age 30, got %d", u.Age)
	}
	if u.Score != 9.5 {
		t.Errorf("expected score 9.5, got %v", u.Score)
	}
}

func TestMapToStructRecordIDField(t *testing.T) {
	type withRecordID struct {
		ID models.RecordID `json:"id"`
	}
	raw := map[string]any{
		"id": models.RecordID{Table: "users", ID: "abc123"},
	}
	var v withRecordID
	if err := mapToStruct(raw, &v); err != nil {
		t.Fatalf("mapToStruct failed: %v", err)
	}
	if v.ID.Table != "users" || v.ID.ID != "abc123" {
		t.Errorf("expected RecordID users:abc123, got %+v", v.ID)
	}
}

func TestMapToStructNestedLink(t *testing.T) {
	type post struct {
		ID      string `json:"id"`
		Title   string `json:"title"`
		OwnerID string `json:"owner_id"`
	}
	raw := map[string]any{
		"id":       models.RecordID{Table: "posts", ID: "p1"},
		"title":    "hello",
		"owner_id": models.RecordID{Table: "users", ID: "u1"},
	}
	var p post
	if err := mapToStruct(raw, &p); err != nil {
		t.Fatalf("mapToStruct failed: %v", err)
	}
	if p.ID != "posts:p1" {
		t.Errorf("expected posts:p1, got %q", p.ID)
	}
	if p.OwnerID != "users:u1" {
		t.Errorf("expected users:u1, got %q", p.OwnerID)
	}
}

func TestApplyTimestamps(t *testing.T) {
	meta := reflectMeta(reflect.TypeOf(testUser{}))
	u := testUser{}
	attrs := map[string]any{}
	applyTimestamps(attrs, &u, true, meta)
	created, ok := attrs["created_at"].(time.Time)
	if !ok || created.IsZero() {
		t.Error("created_at should be set to a time.Time")
	}
	updated, ok := attrs["updated_at"].(time.Time)
	if !ok || updated.IsZero() {
		t.Error("updated_at should be set to a time.Time")
	}
	if u.CreatedAt.IsZero() || u.UpdatedAt.IsZero() {
		t.Error("struct timestamp fields should be populated")
	}

	preserved := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	u2 := testUser{CreatedAt: preserved}
	attrs2 := map[string]any{"created_at": preserved}
	applyTimestamps(attrs2, &u2, true, meta)
	if !attrs2["created_at"].(time.Time).Equal(preserved) {
		t.Error("existing created_at should not be overwritten")
	}
}

func TestFilterAttrs(t *testing.T) {
	m := Model[testUser]{meta: reflectMeta(reflect.TypeOf(testUser{}))}
	in := map[string]any{"email": "x@y.com", "name": "should-drop"}
	got := m.filterAttrs(in)
	if _, ok := got["email"]; !ok {
		t.Error("fillable email should be kept")
	}
	if _, ok := got["name"]; ok {
		t.Error("non-fillable name should be dropped")
	}
}

func TestFilterAttrsGuarded(t *testing.T) {
	m := Model[testUser]{meta: reflectMeta(reflect.TypeOf(testUser{}))}
	in := map[string]any{"email": "x@y.com", "password": "secret"}
	got := m.filterAttrs(in)
	if _, ok := got["password"]; ok {
		t.Error("guarded password should be dropped")
	}
	if _, ok := got["email"]; !ok {
		t.Error("email should be kept")
	}
}

func TestInsertWhereClause(t *testing.T) {
	cases := []struct {
		sql, want string
	}{
		{"SELECT * FROM users", "SELECT * FROM users WHERE deleted_at IS NONE"},
		{"SELECT * FROM users WHERE age > 5", "SELECT * FROM users WHERE deleted_at IS NONE AND age > 5"},
		{"SELECT * FROM users ORDER BY name LIMIT 5", "SELECT * FROM users WHERE deleted_at IS NONE ORDER BY name LIMIT 5"},
		{"SELECT * FROM users LIMIT 5", "SELECT * FROM users WHERE deleted_at IS NONE LIMIT 5"},
		{"SELECT * FROM users FETCH posts", "SELECT * FROM users WHERE deleted_at IS NONE FETCH posts"},
		{"SELECT * FROM ONLY users", "SELECT * FROM ONLY users WHERE deleted_at IS NONE"},
	}
	for _, c := range cases {
		got := insertWhereClause(c.sql, "deleted_at IS NONE")
		if got != c.want {
			t.Errorf("insertWhereClause(%q) = %q, want %q", c.sql, got, c.want)
		}
	}
}

func TestSoftDeleteSQL(t *testing.T) {
	sql := Query[testUser](nil, "users").ToSQL()
	if !strings.Contains(sql, "deleted_at IS NONE") {
		t.Errorf("default query should filter trashed, got: %s", sql)
	}

	sql = Query[testUser](nil, "users").WithTrashed().ToSQL()
	if strings.Contains(sql, "deleted_at") {
		t.Errorf("WithTrashed should remove the filter, got: %s", sql)
	}

	sql = Query[testUser](nil, "users").OnlyTrashed().ToSQL()
	if !strings.Contains(sql, "deleted_at IS NOT NONE") {
		t.Errorf("OnlyTrashed should filter to trashed only, got: %s", sql)
	}

	sql = Query[any](nil, "users").ToSQL()
	if strings.Contains(sql, "deleted_at") {
		t.Errorf("plain query without soft delete field should be untouched, got: %s", sql)
	}
}

func TestWhereInlinesOperator(t *testing.T) {
	sql := Query[any](nil, "users").Where("age", ">", 18).ToSQL()
	if !strings.Contains(sql, "age > 18") {
		t.Errorf("Where should bind the value, got: %s", sql)
	}
}

func TestWhereMirrorsConditions(t *testing.T) {
	model := Query[testUser](nil, "users").WhereEq("active", true).Where("age", ">=", 21)
	if len(model.state.conds) != 2 {
		t.Fatalf("expected 2 mirrored conditions, got %d", len(model.state.conds))
	}
	if model.state.conds[0].sql != "type::field(?) = ?" {
		t.Errorf("unexpected first condition: %q", model.state.conds[0].sql)
	}
}

func TestToIntConversions(t *testing.T) {
	if toInt(int64(42)) != 42 {
		t.Error("toInt(int64) failed")
	}
	if toInt(float64(42.9)) != 42 {
		t.Error("toInt(float64) failed")
	}
}

func TestGetID(t *testing.T) {
	u := testUser{ID: "users:abc"}
	id, ok := getID(&u)
	if !ok || id != "users:abc" {
		t.Errorf("getID failed: %v %v", id, ok)
	}
	empty := testUser{}
	if _, ok := getID(&empty); ok {
		t.Error("getID should report no id for zero value")
	}
}
