package surrealgoorm

import (
	"context"
	"fmt"
	"time"

	"github.com/surrealdb/surrealdb.go"
)

type Config struct {
	URL       string
	Username  string
	Password  string
	Namespace *string
	Database  *string
	MaxConns  int
	Timeout   time.Duration
}
type Option func(*Config)

type DB struct {
	raw    *surrealdb.DB
	tx     *surrealdb.Transaction
	config Config

	queryOverride  func(context.Context, string, map[string]any) (*[]surrealdb.QueryResult[[]map[string]any], error)
	execOverride   func(context.Context, string, map[string]any) (*[]surrealdb.QueryResult[any], error)
	selectOverride func(context.Context, any) (any, error)
	createOverride func(context.Context, any, any) (any, error)
	updateOverride func(context.Context, any, any) (any, error)
	deleteOverride func(context.Context, any) (any, error)
}

func WithMaxConns(n int) Option {
	return func(c *Config) {
		c.MaxConns = n
	}
}
func WithTimeout(d time.Duration) Option {
	return func(c *Config) {
		c.Timeout = d
	}
}

func Connect(ctx context.Context, cfg Config, opts ...Option) (*DB, error) {
	for opt := range opts {
		opts[opt](&cfg)
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = time.Second * 30
	}
	db, err := surrealdb.FromEndpointURLString(ctx, cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to surrealdb: %w", err)
	}
	if _, err := db.SignIn(ctx, surrealdb.Auth{
		Username: cfg.Username,
		Password: cfg.Password,
	}); err != nil {
		return nil, fmt.Errorf("failed to authenticate to surrealdb: %w ", err)
	}
	if cfg.Namespace != nil && cfg.Database != nil {
		if err := db.Use(ctx, *cfg.Namespace, *cfg.Database); err != nil {
			return nil, fmt.Errorf("failed to connect to namespace %s and database %s : %w", *cfg.Namespace, *cfg.Database, err)
		}
	}

	return &DB{
		raw:    db,
		config: cfg,
	}, nil
}

func (db *DB) Raw() *surrealdb.DB {
	return db.raw
}

func (db *DB) Close(ctx context.Context) error {
	if db.raw != nil {
		return db.raw.Close(ctx)
	}
	return nil
}

func (db *DB) Config() Config {
	return db.config
}
func (db *DB) Exec(ctx context.Context, sql string, params map[string]any) error {
	_, err := db.execQuery(ctx, sql, params)
	return err
}

func (db *DB) Begin(ctx context.Context) (*DB, error) {
	if db.raw == nil {
		return nil, ErrInvalidModel
	}
	tx, err := db.raw.Begin(ctx)
	if err != nil {
		return nil, err
	}

	return &DB{
		raw:    db.raw,
		tx:     tx,
		config: db.config,
	}, nil
}
func (db *DB) Commit(ctx context.Context) error {
	if db.tx == nil {
		return ErrNoActiveTx
	}
	return db.tx.Commit(ctx)
}

func (db *DB) Rollback(ctx context.Context) error {
	if db.tx == nil {
		return ErrNoActiveTx
	}
	return db.tx.Cancel(ctx)
}
