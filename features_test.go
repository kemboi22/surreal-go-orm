package surrealgoorm

import (
	"strings"
	"testing"

	"github.com/surrealdb/surrealdb.go/contrib/surrealql"
)

func TestOrderByDescBuildsDescending(t *testing.T) {
	sql := Query[testUser](nil, "user").OrderByDesc("created_at").ToSQL()
	if !strings.Contains(sql, "ORDER BY created_at DESC") {
		t.Fatalf("expected ORDER BY created_at DESC, got %q", sql)
	}
}

func TestOrderByDirectionArgument(t *testing.T) {
	desc := Query[testUser](nil, "user").OrderBy("created_at", "DESC").ToSQL()
	if !strings.Contains(desc, "ORDER BY created_at DESC") {
		t.Fatalf("expected descending order, got %q", desc)
	}
	asc := Query[testUser](nil, "user").OrderBy("name", "ASC").ToSQL()
	if strings.Contains(asc, "DESC") {
		t.Fatalf("expected ascending order, got %q", asc)
	}
}

func TestWhereContainsBuildsClause(t *testing.T) {
	sql, vars := Query[testUser](nil, "user").WhereContains("tags", "vip").ToBuildSQL()
	if !strings.Contains(sql, "tags CONTAINS $") {
		t.Fatalf("expected CONTAINS clause, got %q", sql)
	}
	if len(vars) != 1 {
		t.Fatalf("expected 1 bound var, got %d", len(vars))
	}
}

func TestWhereContainsAnyBuildsClause(t *testing.T) {
	sql, vars := Query[testUser](nil, "user").WhereContainsAny("tags", "vip", "lead").ToBuildSQL()
	if !strings.Contains(sql, "tags CONTAINSANY $") {
		t.Fatalf("expected CONTAINSANY clause, got %q", sql)
	}
	if len(vars) != 1 {
		t.Fatalf("expected 1 bound var, got %d", len(vars))
	}
}

func TestWhereContainsAnyEmptyIsNoop(t *testing.T) {
	sql := Query[testUser](nil, "user").WhereContainsAny("tags").ToSQL()
	if strings.Contains(sql, "CONTAINSANY") {
		t.Fatalf("expected no clause for empty values, got %q", sql)
	}
}

func TestIncrementSQL(t *testing.T) {
	m := Query[testUser](nil, "user")
	sql, vars, err := m.incrementSQL(surrealql.Update("user:abc"), "score", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "score += $") {
		t.Fatalf("expected += clause, got %q", sql)
	}
	if !strings.Contains(sql, "updated_at = time::now()") {
		t.Fatalf("expected updated_at bump, got %q", sql)
	}
	if !strings.Contains(sql, "RETURN AFTER") {
		t.Fatalf("expected RETURN AFTER, got %q", sql)
	}
	if len(vars) == 0 {
		t.Fatal("expected a bound increment value")
	}
}

func TestDecrementSQL(t *testing.T) {
	m := Query[testUser](nil, "user")
	sql, _, err := m.incrementSQL(surrealql.Update("user:abc"), "score", -3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sql, "score -= $") {
		t.Fatalf("expected -= clause, got %q", sql)
	}
}

func TestIncrementRejectsInvalidColumn(t *testing.T) {
	m := Query[testUser](nil, "user")
	if _, _, err := m.incrementSQL(surrealql.Update("user:abc"), "score; DROP TABLE user", 1); err == nil {
		t.Fatal("expected error for invalid column name")
	}
}

func TestValidIdentifier(t *testing.T) {
	for _, name := range []string{"credits", "updated_at", "_x", "a1"} {
		if !validIdentifier(name) {
			t.Errorf("expected %q to be a valid identifier", name)
		}
	}
	for _, name := range []string{"", "1a", "a b", "a;b", "a-b", "a.b"} {
		if validIdentifier(name) {
			t.Errorf("expected %q to be an invalid identifier", name)
		}
	}
}
