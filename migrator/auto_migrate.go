package migrator

import (
	"context"

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
	for _, m := range models {
		if err := m.Up(ctx, *schema); err != nil {
			return err
		}
	}
	return nil
}
