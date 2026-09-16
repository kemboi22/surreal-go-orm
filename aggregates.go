package surrealgoorm

import (
	"context"
)

func (m Model[T]) Count(ctx context.Context) (int, error) {
	val, err := m.aggregate(ctx, "count()")
	if err != nil {
		return 0, err
	}
	return toInt(val), nil
}

func (m Model[T]) Exists(ctx context.Context) (bool, error) {
	n, err := m.Count(ctx)
	return n > 0, err
}

func (m Model[T]) Sum(ctx context.Context, column string) (float64, error) {
	return m.mathAggregate(ctx, "math::sum("+column+")")
}

func (m Model[T]) Avg(ctx context.Context, column string) (float64, error) {
	return m.mathAggregate(ctx, "math::avg("+column+")")
}

func (m Model[T]) Min(ctx context.Context, column string) (float64, error) {
	return m.mathAggregate(ctx, "math::min("+column+")")
}

func (m Model[T]) Max(ctx context.Context, column string) (float64, error) {
	return m.mathAggregate(ctx, "math::max("+column+")")
}

func (m Model[T]) mathAggregate(ctx context.Context, fn string) (float64, error) {
	val, err := m.aggregate(ctx, fn)
	if err != nil {
		return 0, err
	}
	if val == nil {
		return 0, nil
	}
	if f, ok := toFloat64(val); ok {
		return f, nil
	}
	return 0, nil
}

func (m Model[T]) aggregate(ctx context.Context, fn string) (any, error) {
	q := m.baseSelect().Alias("_agg", fn).GroupAll()
	sql, vars := q.Build()
	sql = m.applyTrash(sql)
	rows, err := m.queryMaps(ctx, sql, vars)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0]["_agg"], nil
}
