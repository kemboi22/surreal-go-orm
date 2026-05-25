package surrealgoorm

import (
	"context"

	"github.com/surrealdb/surrealdb.go"
)

type Migration interface {
	Name() string
	Up(ctx context.Context, db *surrealdb.DB) error
	Down(ctx context.Context, db *surrealdb.DB) error
}
