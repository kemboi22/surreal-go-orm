package orm

import (
	"context"
	"fmt"

	"github.com/surrealdb/surrealdb.go"
)

type DB struct {
	Raw *surrealdb.DB
}
type Config struct {
	URL       string
	Username  string
	Password  string
	Namespace *string
	Database  *string
}

func Connect(ctx context.Context, cfg Config) (*DB, error) {
	db, err := surrealdb.FromEndpointURLString(ctx, cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to surrealdb at %s: %w", cfg.URL, err)
	}
	if _, err = db.SignIn(ctx, map[string]any{
		"user":     cfg.Username,
		"password": cfg.Password,
	}); err != nil {
		return nil, fmt.Errorf("surrealdb authentication failed: %w", err)
	}

	// Use Namespace && Database if it exists
	if cfg.Namespace != nil && cfg.Database != nil {
		if err = db.Use(ctx, *cfg.Namespace, *cfg.Database); err != nil {
			return nil, fmt.Errorf("failed to use namespace %s and database %s : %w", *cfg.Namespace, *cfg.Database, err)
		}
	}

	return &DB{
		Raw: db,
	}, nil
}

func (db *DB) Close(ctx context.Context) error {
	if db.Raw != nil {
		return db.Raw.Close(ctx)
	}
	return nil
}
