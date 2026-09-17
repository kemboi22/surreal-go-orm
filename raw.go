package surrealgoorm

import (
	"context"
	"fmt"

	"github.com/surrealdb/surrealdb.go"
)

// Raw executes raw SurrealQL against a database connection and checks every
// statement for errors, including statements after the first. It is an escape
// hatch for DDL, aggregates, nested subqueries and atomic expressions that the
// query builder cannot express; prefer the builder for everything else.
func Raw[T any](ctx context.Context, db *surrealdb.DB, sql string, vars map[string]any) (*[]surrealdb.QueryResult[T], error) {
	results, err := surrealdb.Query[T](ctx, db, sql, vars)
	if err != nil {
		return nil, err
	}
	if results != nil {
		for i, result := range *results {
			if result.Error != nil {
				return nil, fmt.Errorf("surrealgoorm: query statement %d: %w", i+1, result.Error)
			}
			if result.Status == "ERR" {
				return nil, fmt.Errorf("surrealgoorm: query statement %d failed", i+1)
			}
		}
	}
	return results, nil
}

// Exec runs raw SurrealQL and discards the result.
func Exec(ctx context.Context, db *surrealdb.DB, sql string, vars map[string]any) error {
	_, err := Raw[any](ctx, db, sql, vars)
	return err
}
