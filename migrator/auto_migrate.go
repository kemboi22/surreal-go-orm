package migrator

import (
	"context"
	"errors"
	"fmt"

	surrealgoorm "github.com/kemboi22/surreal-go-orm"
	"github.com/surrealdb/surrealdb.go"
)

type AutoMigrator struct {
	Db *surrealdb.DB
}

func NewAutoMigrator(db *surrealdb.DB) *AutoMigrator {
	return &AutoMigrator{
		Db: db,
	}
}

func (migrator *AutoMigrator) AutoMigrate(ctx context.Context, models []surrealgoorm.Migration) error {
	schema := &surrealgoorm.Schema{
		Db: migrator.Db,
	}
	var errs error
	for _, m := range models {
		if err := m.Up(ctx, *schema); err != nil {
			errs = errors.Join(errs, fmt.Errorf("%s: %w", m.Name(), err))
		}
	}
	return errs
}
