package surrealgoorm

import (
	"strings"
	"testing"
)

func TestTableID(t *testing.T) {
	table := &Table{Name: "users"}
	col := table.ID()
	if col.Name != "id" {
		t.Errorf("ID() column name should be 'id', got '%s'", col.Name)
	}
	if col.Type != "record<users>" {
		t.Errorf("ID() column type should be 'record<users>', got '%s'", col.Type)
	}
	if len(table.Columns) != 1 {
		t.Errorf("expected 1 column, got %d", len(table.Columns))
	}
}

func TestTableString(t *testing.T) {
	table := &Table{Name: "users"}
	col := table.String("name")
	if col.Name != "name" || col.Type != "string" {
		t.Errorf("String() created wrong column: %+v", col)
	}
}

func TestTableInt(t *testing.T) {
	table := &Table{Name: "users"}
	col := table.Int("age")
	if col.Name != "age" || col.Type != "int" {
		t.Errorf("Int() created wrong column: %+v", col)
	}
}

func TestTableBool(t *testing.T) {
	table := &Table{Name: "users"}
	col := table.Bool("active")
	if col.Name != "active" || col.Type != "boolean" {
		t.Errorf("Bool() created wrong column: %+v", col)
	}
}

func TestTableArray(t *testing.T) {
	table := &Table{Name: "users"}
	col := table.Array("tags")
	if col.Name != "tags" || col.Type != "array" {
		t.Errorf("Array() created wrong column: %+v", col)
	}
}

func TestTableBytes(t *testing.T) {
	table := &Table{Name: "users"}
	col := table.Bytes("data")
	if col.Name != "data" || col.Type != "bytes" {
		t.Errorf("Bytes() created wrong column: %+v", col)
	}
}

func TestTableDatetime(t *testing.T) {
	table := &Table{Name: "users"}
	col := table.Datetime("created_at")
	if col.Name != "created_at" || col.Type != "datetime" {
		t.Errorf("Datetime() created wrong column: %+v", col)
	}
}

func TestTableDecimal(t *testing.T) {
	table := &Table{Name: "users"}
	col := table.Decimal("price")
	if col.Name != "price" || col.Type != "decimal" {
		t.Errorf("Decimal() created wrong column: %+v", col)
	}
}

func TestTableDuration(t *testing.T) {
	table := &Table{Name: "users"}
	col := table.Duration("timeout")
	if col.Name != "timeout" || col.Type != "duration" {
		t.Errorf("Duration() created wrong column: %+v", col)
	}
}

func TestTableFloat(t *testing.T) {
	table := &Table{Name: "users"}
	col := table.Float("rating")
	if col.Name != "rating" || col.Type != "float" {
		t.Errorf("Float() created wrong column: %+v", col)
	}
}

func TestTableGeometry(t *testing.T) {
	table := &Table{Name: "places"}
	col := table.Geometry("location", "point")
	if col.Name != "location" || col.Type != "geometry<point>" {
		t.Errorf("Geometry() created wrong column: %+v", col)
	}
}

func TestTableNumber(t *testing.T) {
	table := &Table{Name: "users"}
	col := table.Number("score")
	if col.Name != "score" || col.Type != "number" {
		t.Errorf("Number() created wrong column: %+v", col)
	}
}

func TestTableObject(t *testing.T) {
	table := &Table{Name: "users"}
	col := table.Object("metadata")
	if col.Name != "metadata" || col.Type != "object" {
		t.Errorf("Object() created wrong column: %+v", col)
	}
}

func TestTableRegex(t *testing.T) {
	table := &Table{Name: "users"}
	col := table.Regex("pattern")
	if col.Name != "pattern" || col.Type != "regex" {
		t.Errorf("Regex() created wrong column: %+v", col)
	}
}

func TestTableTimestamps(t *testing.T) {
	table := &Table{Name: "users"}
	table.Timestamps()
	if len(table.Columns) != 2 {
		t.Fatalf("expected 2 timestamp columns, got %d", len(table.Columns))
	}
	if table.Columns[0].Name != "created_at" || table.Columns[0].Type != "datetime" {
		t.Errorf("first column should be created_at datetime, got %+v", table.Columns[0])
	}
	if table.Columns[1].Name != "updated_at" || table.Columns[1].Type != "datetime" {
		t.Errorf("second column should be updated_at datetime, got %+v", table.Columns[1])
	}
}

func TestTableBuildSimple(t *testing.T) {
	table := &Table{Name: "users"}
	table.String("name")
	table.Int("age")

	got := table.Build()
	if !strings.Contains(got, "DEFINE TABLE users SCHEMAFULL") {
		t.Error("Build() should contain table definition")
	}
	if !strings.Contains(got, "DEFINE FIELD name ON users type string;") {
		t.Error("Build() should contain name field")
	}
	if !strings.Contains(got, "DEFINE FIELD age ON users type int;") {
		t.Error("Build() should contain age field")
	}
}

func TestTableBuildWithUnique(t *testing.T) {
	table := &Table{Name: "users"}
	table.String("email").Unique()
	table.String("name")

	got := table.Build()
	if !strings.Contains(got, "DEFINE INDEX email_unique") {
		t.Error("Build() should contain unique index for email")
	}
	if !strings.Contains(got, "ON users") {
		t.Error("Build() should reference the table in the index")
	}
	if !strings.Contains(got, "FIELDS email UNIQUE;") {
		t.Error("Build() should mark the index as UNIQUE")
	}
}

func TestTableBuildWithNullable(t *testing.T) {
	table := &Table{Name: "users"}
	table.String("nickname").Nullable()

	got := table.Build()
	if !strings.Contains(got, "DEFINE FIELD nickname ON users type string;") {
		t.Error("Build() should not add FLEXIBLE for scalar nullable columns")
	}
}

func TestTableBuildWithDefaultString(t *testing.T) {
	table := &Table{Name: "users"}
	table.String("status").Default("active")

	got := table.Build()
	if !strings.Contains(got, "DEFAULT 'active'") {
		t.Errorf("Build() should include DEFAULT with quoted string, got:\n%s", got)
	}
}

func TestTableBuildWithDefaultInt(t *testing.T) {
	table := &Table{Name: "users"}
	table.Int("age").Default(18)

	got := table.Build()
	if !strings.Contains(got, "DEFAULT 18") {
		t.Errorf("Build() should include DEFAULT with int, got:\n%s", got)
	}
}

func TestTableBuildWithDefaultBool(t *testing.T) {
	table := &Table{Name: "users"}
	table.Bool("active").Default(true)
	table.Bool("deleted").Default(false)

	got := table.Build()
	if !strings.Contains(got, "DEFAULT true") {
		t.Errorf("Build() should include DEFAULT true, got:\n%s", got)
	}
	if !strings.Contains(got, "DEFAULT false") {
		t.Errorf("Build() should include DEFAULT false, got:\n%s", got)
	}
}

func TestTableBuildAllModifiers(t *testing.T) {
	table := &Table{Name: "products"}
	table.String("sku").Unique().Nullable().Default("N/A")
	table.Object("meta").Nullable()

	got := table.Build()
	if !strings.Contains(got, "DEFINE INDEX sku_unique") {
		t.Error("Build() should contain unique index")
	}
	if !strings.Contains(got, "type object FLEXIBLE") {
		t.Error("Build() should contain FLEXIBLE for object fields")
	}
	if !strings.Contains(got, "DEFAULT 'N/A'") {
		t.Error("Build() should contain DEFAULT")
	}
}

func TestTableBuildColumnOrder(t *testing.T) {
	table := &Table{Name: "users"}
	c1 := table.String("first")
	c2 := table.Int("second")

	got := table.Build()
	if strings.Index(got, c1.Name) > strings.Index(got, c2.Name) {
		t.Error("Columns should appear in the order they were added")
	}
}

func TestFormatValueString(t *testing.T) {
	got := formatValue("hello")
	want := "'hello'"
	if got != want {
		t.Errorf("formatValue('hello') = '%s', want '%s'", got, want)
	}
}

func TestFormatValueBoolTrue(t *testing.T) {
	got := formatValue(true)
	if got != "true" {
		t.Errorf("formatValue(true) = '%s', want 'true'", got)
	}
}

func TestFormatValueBoolFalse(t *testing.T) {
	got := formatValue(false)
	if got != "false" {
		t.Errorf("formatValue(false) = '%s', want 'false'", got)
	}
}

func TestFormatValueInt(t *testing.T) {
	got := formatValue(42)
	if got != "42" {
		t.Errorf("formatValue(42) = '%s', want '42'", got)
	}
}

func TestFormatValueFloat(t *testing.T) {
	got := formatValue(3.14)
	if got != "3.14" {
		t.Errorf("formatValue(3.14) = '%s', want '3.14'", got)
	}
}

func TestFormatValueStringWithQuotes(t *testing.T) {
	got := formatValue("it's")
	want := "'it's'"
	if got != want {
		t.Errorf("formatValue(\"it's\") = '%s', want '%s'", got, want)
	}
}

func TestTableEmptyName(t *testing.T) {
	table := &Table{Name: ""}
	table.String("col")
	got := table.Build()
	if !strings.Contains(got, "DEFINE TABLE  SCHEMAFULL") {
		t.Error("Build() should handle empty table name")
	}
}

func TestTableNoColumns(t *testing.T) {
	table := &Table{Name: "empty"}
	got := table.Build()
	if got != "DEFINE TABLE empty SCHEMAFULL;" {
		t.Errorf("Build() with no columns should only have table definition, got:\n%s", got)
	}
}
