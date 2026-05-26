package surrealgoorm

import (
	"strings"
	"testing"
)

func TestQueryCreatesModel(t *testing.T) {
	qb := Query[any](nil, "users")
	model, ok := qb.(*Model[any])
	if !ok {
		t.Fatal("Query() should return a *Model")
	}
	if model.table != "users" {
		t.Errorf("expected table 'users', got '%s'", model.table)
	}
}

func TestQueryDefaultToSQL(t *testing.T) {
	qb := Query[any](nil, "users")
	sql := qb.ToSQL()
	if sql == "" {
		t.Error("ToSQL() should not return empty string")
	}
	if !strings.Contains(sql, "SELECT") {
		t.Error("ToSQL() should contain SELECT")
	}
	if !strings.Contains(sql, "users") {
		t.Error("ToSQL() should contain table name")
	}
}

func TestQuerySelect(t *testing.T) {
	qb := Query[any](nil, "users")
	qb = qb.Select("name", "email")
	sql := qb.ToSQL()
	if !strings.Contains(sql, "name") || !strings.Contains(sql, "email") {
		t.Error("ToSQL() with Select() should contain selected fields")
	}
}

func TestQueryWhereEq(t *testing.T) {
	qb := Query[any](nil, "users")
	qb = qb.WhereEq("name", "John")
	sql := qb.ToSQL()
	if !strings.Contains(sql, "WHERE") && !strings.Contains(sql, "where") {
		t.Error("ToSQL() with WhereEq() should contain WHERE clause")
	}
}

func TestQueryWhere(t *testing.T) {
	qb := Query[any](nil, "users")
	qb = qb.Where("age", ">", 18)
	sql := qb.ToSQL()
	if !strings.Contains(sql, "WHERE") && !strings.Contains(sql, "where") {
		t.Error("ToSQL() with Where() should contain WHERE clause")
	}
	if !strings.Contains(sql, ">") {
		t.Error("ToSQL() with Where() should contain the operator")
	}
}

func TestQueryWhereNotNull(t *testing.T) {
	qb := Query[any](nil, "users")
	qb = qb.WhereNotNull("email")
	sql := qb.ToSQL()
	if !strings.Contains(sql, "NOT") || !strings.Contains(sql, "NULL") {
		t.Error("ToSQL() with WhereNotNull() should contain NOT NULL")
	}
}

func TestQueryWhereNull(t *testing.T) {
	qb := Query[any](nil, "users")
	qb = qb.WhereNull("deleted_at")
	sql := qb.ToSQL()
	if !strings.Contains(sql, "NULL") {
		t.Error("ToSQL() with WhereNull() should contain NULL")
	}
}

func TestQueryOrderBy(t *testing.T) {
	qb := Query[any](nil, "users")
	qb = qb.OrderBy("name")
	sql := qb.ToSQL()
	if !strings.Contains(sql, "ORDER BY") && !strings.Contains(sql, "order by") {
		t.Error("ToSQL() with OrderBy() should contain ORDER BY")
	}
}

func TestQueryLimit(t *testing.T) {
	qb := Query[any](nil, "users")
	qb = qb.Limit(10)
	sql := qb.ToSQL()
	if !strings.Contains(sql, "LIMIT") && !strings.Contains(sql, "limit") {
		t.Error("ToSQL() with Limit() should contain LIMIT")
	}
}

func TestQueryWith(t *testing.T) {
	qb := Query[any](nil, "users")
	sqlBefore := qb.ToSQL()
	qb = qb.With("posts", "comments")
	sqlAfter := qb.ToSQL()
	if sqlBefore == sqlAfter {
		t.Error("ToSQL() with With() should modify the query")
	}
}

func TestQueryMethodChaining(t *testing.T) {
	qb := Query[any](nil, "users")
	sql := qb.WhereEq("active", true).OrderBy("name").Limit(5).ToSQL()
	if !strings.Contains(sql, "WHERE") && !strings.Contains(sql, "where") {
		t.Error("Chained WhereEq should affect SQL")
	}
	if !strings.Contains(sql, "ORDER BY") && !strings.Contains(sql, "order by") {
		t.Error("Chained OrderBy should affect SQL")
	}
	if !strings.Contains(sql, "LIMIT") && !strings.Contains(sql, "limit") {
		t.Error("Chained Limit should affect SQL")
	}
}

func TestQueryToBuildSQL(t *testing.T) {
	qb := Query[any](nil, "users")
	qb = qb.WhereEq("name", "John")

	sql, vars := qb.ToBuildSQL()
	if sql == "" {
		t.Error("ToBuildSQL() should return non-empty SQL")
	}
	if len(vars) == 0 {
		t.Log("ToBuildSQL() returned 0 vars - may be valid if var is inline")
	}
}

func TestQueryMultipleWhereClauses(t *testing.T) {
	qb := Query[any](nil, "users")
	qb = qb.WhereEq("active", true).Where("age", ">=", 21)
	sql := qb.ToSQL()
	if !strings.Contains(sql, "active") {
		t.Error("SQL should contain 'active'")
	}
	if !strings.Contains(sql, "age") {
		t.Error("SQL should contain 'age'")
	}
}


