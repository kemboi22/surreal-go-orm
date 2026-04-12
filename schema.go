package surrealgoorm

import (
	"context"
	"fmt"
	"strings"

	ormtypes "github.com/kemboi22/surreal-go-orm/types"
)

// TableSchema is a small builder for SurrealDB table and field SQL.
type TableSchema struct {
	Name   string
	Schema string
	Fields []ormtypes.SchemaField
}

func NewTableSchema(name string) *TableSchema {
	return &TableSchema{Name: name}
}

func (s *TableSchema) AddField(field ormtypes.SchemaField) *TableSchema {
	s.Fields = append(s.Fields, field)
	return s
}

// / Schema
// / Either SCHEMALESS OR SCHEMAFULL
func (s *TableSchema) SchemaType(schema string) *TableSchema {
	s.Schema = schema
	return s
}
func (s *TableSchema) SchemaFull() *TableSchema {
	s.Schema = "SCHEMAFULL"
	return s
}
func (s *TableSchema) SchemaLess() *TableSchema {
	s.Schema = "SCHEMALESS"
	return s
}

func (s *TableSchema) Statements() ([]string, error) {
	if strings.TrimSpace(s.Name) == "" {
		return nil, ErrEmptyTableName
	}
	if strings.TrimSpace(s.Schema) == "" {
		return nil, ErrEmptySchemaType
	}
	if s.Schema != "SCHEMAFULL" && s.Schema != "SCHEMALESS" {
		return nil, ErrInvalidSchemaType
	}

	statements := []string{
		fmt.Sprintf("DEFINE TABLE %s %s;", s.Name, s.Schema),
	}

	for _, field := range s.Fields {
		fieldSQL := fmt.Sprintf("DEFINE FIELD %s ON %s TYPE %s;", field.Name, s.Name, field.Type)
		statements = append(statements, fieldSQL)

		for _, option := range field.Options {
			if option.Default != nil {
				statements = append(statements, fmt.Sprintf("DEFINE FIELD %s ON %s DEFAULT %s TYPE %s;", field.Name, s.Name, formatSchemaLiteral(option.Default), field.Type))
			}
			if option.Index || option.Unique {
				indexName := fmt.Sprintf("%s_%s_idx", s.Name, field.Name)
				unique := ""
				if option.Unique {
					unique = " UNIQUE"
				}
				statements = append(statements, fmt.Sprintf("DEFINE INDEX %s ON %s FIELDS %s%s;", indexName, s.Name, field.Name, unique))
			}
		}
	}

	return statements, nil
}

func formatSchemaLiteral(value any) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("%q", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", value)
	}
}

type Migration interface {
	Name() string
	Up() ([]string, error)
	Down() ([]string, error)
}

type SQLMigration struct {
	MigrationName string
	UpSQL         []string
	DownSQL       []string
}

func (m SQLMigration) Name() string            { return m.MigrationName }
func (m SQLMigration) Up() ([]string, error)   { return m.UpSQL, nil }
func (m SQLMigration) Down() ([]string, error) { return m.DownSQL, nil }

type Migrator struct {
	db    *DB
	table string
}

func NewMigrator(db *DB) *Migrator {
	return &Migrator{
		db:    db,
		table: "_orm_migrations",
	}
}

func (m *Migrator) EnsureTable(ctx context.Context) error {
	sql := fmt.Sprintf("DEFINE TABLE %s SCHEMAFULL; DEFINE FIELD name ON %s TYPE string; DEFINE INDEX %s_name_idx ON %s FIELDS name UNIQUE;", m.table, m.table, m.table, m.table)
	return m.db.Exec(ctx, sql, nil)
}

func (m *Migrator) Migrate(ctx context.Context, migrations ...Migration) error {
	if err := m.EnsureTable(ctx); err != nil {
		return err
	}

	for _, migration := range migrations {
		exists, err := m.applied(ctx, migration.Name())
		if err != nil {
			return err
		}
		if exists {
			continue
		}

		statements, err := migration.Up()
		if err != nil {
			return err
		}
		for _, statement := range statements {
			if err := m.db.Exec(ctx, statement, nil); err != nil {
				return err
			}
		}

		if err := m.db.Exec(ctx, fmt.Sprintf("CREATE %s SET name = $name", m.table), map[string]any{"name": migration.Name()}); err != nil {
			return err
		}
	}

	return nil
}

func (m *Migrator) Rollback(ctx context.Context, migration Migration) error {
	statements, err := migration.Down()
	if err != nil {
		return err
	}

	for _, statement := range statements {
		if err := m.db.Exec(ctx, statement, nil); err != nil {
			return err
		}
	}

	deleteSQL := fmt.Sprintf("DELETE FROM %s WHERE name = $name", m.table)
	return m.db.Exec(ctx, deleteSQL, map[string]any{"name": migration.Name()})
}

func (m *Migrator) AutoMigrate(ctx context.Context, schemas ...*TableSchema) error {
	for _, schema := range schemas {
		stmts, err := schema.Statements()
		if err != nil {
			return err
		}
		migration := schemaMigration{
			name:    fmt.Sprintf("auto_%s", schema.Name),
			upSQL:   stmts,
			downSQL: []string{fmt.Sprintf("DROP TABLE %s", schema.Name)},
		}
		if err := m.Migrate(ctx, migration); err != nil {
			return err
		}
	}
	return nil
}

type schemaMigration struct {
	name    string
	upSQL   []string
	downSQL []string
}

func (m schemaMigration) Name() string            { return m.name }
func (m schemaMigration) Up() ([]string, error)   { return m.upSQL, nil }
func (m schemaMigration) Down() ([]string, error) { return m.downSQL, nil }

func (m *Migrator) applied(ctx context.Context, name string) (bool, error) {
	var rows []map[string]any
	sql := fmt.Sprintf("SELECT * FROM %s WHERE name = $name LIMIT 1", m.table)
	if err := m.db.RawQuery(sql).All(ctx, &rows); err != nil {
		return false, err
	}
	return len(rows) > 0, nil
}
