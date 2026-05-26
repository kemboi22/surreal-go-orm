package surrealgoorm

import (
	"context"

	"github.com/surrealdb/surrealdb.go"
)

type Schema struct {
	Db *surrealdb.DB
}

func (s Schema) CreateTable(ctx context.Context, name string, fn func(*Table)) error {
	table := &Table{
		Name: name,
	}
	fn(table)

	return nil
}
