package surrealgoorm

import (
	"context"

	"github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/contrib/surrealql"
)

type QueryBuilder[T any] interface {
	Select(columns ...string) QueryBuilder[T]
	Where(column, operator string, value any) QueryBuilder[T]
	WhereEq(column string, value any) QueryBuilder[T]
	WhereNotNull(column string) QueryBuilder[T]
	WhereNull(column string) QueryBuilder[T]
	OrderBy(column string) QueryBuilder[T]
	Limit(limit int) QueryBuilder[T]
	ToSQL() string
	First(ctx context.Context) (*T, error)
	Get(ctx context.Context) (*[]T, error)
}

type Model[T any] struct {
	table string
	sq    *surrealql.SelectQuery
	db    *surrealdb.DB
}

func Query[T any](db *surrealdb.DB, table string) QueryBuilder[T] {
	return &Model[T]{
		table: table,
		sq:    surrealql.Select(table),
		db:    db,
	}
}

func (m Model[T]) Select(columns ...string) QueryBuilder[T] {
	m.sq.Fields(columns)
	return m
}

func (m Model[T]) Where(column, operator string, value any) QueryBuilder[T] {
	m.sq.Where(column+" "+operator, value)
	return m
}

func (m Model[T]) WhereEq(column string, value any) QueryBuilder[T] {
	m.sq.WhereEq(column, value)
	return m
}
func (m Model[T]) WhereNotNull(column string) QueryBuilder[T] {
	m.sq.WhereNotNull(column)
	return m
}
func (m Model[T]) WhereNull(column string) QueryBuilder[T] {
	m.sq.WhereNull(column)
	return m
}
func (m Model[T]) OrderBy(column string) QueryBuilder[T] {
	m.sq.OrderBy(column)
	return m
}
func (m Model[T]) Limit(limit int) QueryBuilder[T] {
	m.sq.Limit(limit)
	return m
}
func (m Model[T]) ToSQL() string {
	sql, _ := m.sq.Build()
	return sql
}
func (m Model[T]) First(ctx context.Context) (*T, error) {
	sql, vars := m.sq.Build()
	res, err := surrealdb.Query[[]T](ctx, m.db, sql, vars)
	if err != nil {
		return nil, err
	}
	if len((*res)) == 0 || len((*res)[0].Result) == 0 {
		return nil, nil
	}
	return &((*res)[0].Result[0]), nil
}

func (m Model[T]) Get(ctx context.Context) (*[]T, error) {
	sql, vars := m.sq.Build()
	res, err := surrealdb.Query[[]T](ctx, m.db, sql, vars)
	if err != nil {
		return nil, err
	}
	if len((*res)) == 0 {
		t := new([]T)
		return t, nil
	}
	return &(*res)[0].Result, nil
}
