package surrealgoorm

import (
	"context"
	"fmt"

	"github.com/surrealdb/surrealdb.go/contrib/surrealql"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

// Increment atomically adds by to column on the record and returns the updated
// record.
//
//	UPDATE widget:abc SET credits += 5, updated_at = time::now() RETURN AFTER
func (m Model[T]) Increment(ctx context.Context, id any, column string, by int64) (*T, error) {
	return m.adjust(ctx, m.recordID(id), column, by)
}

// Decrement atomically subtracts by from column on the record and returns the
// updated record.
func (m Model[T]) Decrement(ctx context.Context, id any, column string, by int64) (*T, error) {
	return m.adjust(ctx, m.recordID(id), column, -by)
}

// WhereIncrement atomically adds by to column for every record matching the
// current WHERE clauses and returns the updated records.
func (m Model[T]) WhereIncrement(ctx context.Context, column string, by int64) ([]T, error) {
	return m.adjustWhere(ctx, column, by)
}

// WhereDecrement atomically subtracts by from column for every record matching
// the current WHERE clauses and returns the updated records.
func (m Model[T]) WhereDecrement(ctx context.Context, column string, by int64) ([]T, error) {
	return m.adjustWhere(ctx, column, -by)
}

func (m Model[T]) adjust(ctx context.Context, rid models.RecordID, column string, delta int64) (*T, error) {
	sql, vars, err := m.incrementSQL(surrealql.Update(rid), column, delta)
	if err != nil {
		return nil, err
	}
	rows, err := m.queryAll(ctx, sql, vars)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}

func (m Model[T]) adjustWhere(ctx context.Context, column string, delta int64) ([]T, error) {
	q := surrealql.Update(m.table)
	for _, c := range m.state.conds {
		q = q.Where(c.sql, c.args...)
	}
	sql, vars, err := m.incrementSQL(q, column, delta)
	if err != nil {
		return nil, err
	}
	return m.queryAll(ctx, sql, vars)
}

func (m Model[T]) incrementSQL(q *surrealql.UpdateQuery, column string, delta int64) (string, map[string]any, error) {
	if !validIdentifier(column) {
		return "", nil, fmt.Errorf("surrealgoorm: invalid column %q", column)
	}
	if delta < 0 {
		q = q.Set(column+" -= ?", -delta)
	} else {
		q = q.Set(column+" += ?", delta)
	}
	if m.meta != nil && m.meta.timestamps {
		q = q.Set("updated_at = time::now()")
	}
	q = q.Return(surrealql.ReturnAfterClause)
	sql, vars := q.Build()
	return sql, vars, nil
}
