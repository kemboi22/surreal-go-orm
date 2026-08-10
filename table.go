package surrealgoorm

import (
	"fmt"
	"strings"
)

type Table struct {
	Name    string
	Columns []*Column
}

func (t *Table) addColumn(name, typ string) *Column {
	c := &Column{
		Name: name,
		Type: typ,
	}
	t.Columns = append(t.Columns, c)
	return c
}

func (t *Table) ID() *Column {
	return t.addColumn("id", "record<"+t.Name+">")
}

func (t *Table) String(name string) *Column {
	return t.addColumn(name, "string")
}
func (t *Table) Int(name string) *Column {
	return t.addColumn(name, "int")
}

func (t *Table) Bool(name string) *Column {
	return t.addColumn(name, "boolean")
}
func (t *Table) Array(name string) *Column {
	return t.addColumn(name, "array")
}
func (t *Table) Bytes(name string) *Column {
	return t.addColumn(name, "bytes")
}
func (t *Table) Datetime(name string) *Column {
	return t.addColumn(name, "datetime")
}
func (t *Table) Decimal(name string) *Column {
	return t.addColumn(name, "decimal")
}
func (t *Table) Duration(name string) *Column {
	return t.addColumn(name, "duration")
}
func (t *Table) Float(name string) *Column {
	return t.addColumn(name, "float")
}
func (t *Table) Geometry(name, typ string) *Column {
	return t.addColumn(name, "geometry<"+typ+">")
}
func (t *Table) Number(name string) *Column {
	return t.addColumn(name, "number")
}
func (t *Table) Object(name string) *Column {
	return t.addColumn(name, "object")
}
func (t *Table) Regex(name string) *Column {
	return t.addColumn(name, "regex")
}

func (t *Table) Timestamps() {
	t.Datetime("created_at")
	t.Datetime("updated_at")
}

func (t *Table) Build() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("DEFINE TABLE %s SCHEMAFULL;", t.Name))
	for _, col := range t.Columns {
		if col.Name == "id" {
			continue
		}
		colType := col.Type
		if col.Optional && col.DefaultValue == nil && !strings.Contains(col.Type, "object") {
			// Optional field with no default: represent as option<T> so an
			// omitted value is allowed by SurrealDB's SCHEMAFULL fields.
			colType = "option<" + colType + ">"
		}
		field := fmt.Sprintf("DEFINE FIELD %s ON %s type %s", col.Name, t.Name, colType)
		// An optional field with an explicit default keeps that default.
		switch {
		case col.Optional && col.DefaultValue != nil:
			field += fmt.Sprintf(" DEFAULT %v", formatValue(col.DefaultValue))
		case col.Optional && strings.Contains(col.Type, "object"):
			field += " FLEXIBLE"
		case col.DefaultValue != nil:
			field += fmt.Sprintf(" DEFAULT %v", formatValue(col.DefaultValue))
		}
		field += ";"
		parts = append(parts, field)
		if col.IsUnique {
			parts = append(parts, fmt.Sprintf(`
			DEFINE INDEX %s_unique 
			ON %s 
			FIELDS %s UNIQUE;
			`, col.Name, t.Name, col.Name))
		}
	}
	return strings.Join(parts, "\n")
}
func formatValue(v any) string {

	switch val := v.(type) {

	case string:
		return fmt.Sprintf("'%s'", val)

	case bool:
		if val {
			return "true"
		}
		return "false"

	default:
		return fmt.Sprintf("%v", val)
	}
}
