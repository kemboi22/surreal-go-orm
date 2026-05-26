package surrealgoorm

import (
	"context"
)

type Migration interface {
	Name() string
	Up(ctx context.Context, schema Schema) error
	Down(ctx context.Context, schema Schema) error
}
