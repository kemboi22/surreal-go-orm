package surrealgoorm

import (
	"context"
	"errors"
	"time"

	"github.com/surrealdb/surrealdb.go/contrib/surrealql"
)

var ErrNotFound = errors.New("surrealgoorm: record not found")

func (m Model[T]) Create(ctx context.Context, data *T) (*T, error) {
	if err := fireHook(data, hookSaving); err != nil {
		return nil, err
	}
	if err := fireHook(data, hookCreating); err != nil {
		return nil, err
	}
	attrs, err := structToMap(data)
	if err != nil {
		return nil, err
	}
	delete(attrs, "id")
	applyTimestamps(attrs, data, true, m.meta)
	q := surrealql.Create(m.table).Content(attrs).Return(surrealql.ReturnAfterClause)
	sql, vars := q.Build()
	rows, err := m.queryMaps(ctx, sql, vars)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		fireHook(data, hookCreated)
		fireHook(data, hookSaved)
		return data, nil
	}
	if err := mapToStruct(rows[0], data); err != nil {
		return nil, err
	}
	fireHook(data, hookCreated)
	fireHook(data, hookSaved)
	return data, nil
}

func (m Model[T]) CreateWithID(ctx context.Context, id any, data *T) (*T, error) {
	if err := fireHook(data, hookSaving); err != nil {
		return nil, err
	}
	if err := fireHook(data, hookCreating); err != nil {
		return nil, err
	}
	attrs, err := structToMap(data)
	if err != nil {
		return nil, err
	}
	delete(attrs, "id")
	applyTimestamps(attrs, data, true, m.meta)
	q := surrealql.Create(surrealql.Thing(m.table, id)).Content(attrs).Return(surrealql.ReturnAfterClause)
	sql, vars := q.Build()
	rows, err := m.queryMaps(ctx, sql, vars)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		fireHook(data, hookCreated)
		fireHook(data, hookSaved)
		return data, nil
	}
	if err := mapToStruct(rows[0], data); err != nil {
		return nil, err
	}
	fireHook(data, hookCreated)
	fireHook(data, hookSaved)
	return data, nil
}

func (m Model[T]) CreateMany(ctx context.Context, data []T) ([]T, error) {
	items := make([]map[string]any, 0, len(data))
	for i := range data {
		attrs, err := structToMap(&data[i])
		if err != nil {
			return nil, err
		}
		delete(attrs, "id")
		applyTimestamps(attrs, &data[i], true, m.meta)
		items = append(items, attrs)
	}
	q := surrealql.Insert(m.table).Value(items).Return(surrealql.ReturnAfterClause)
	sql, vars := q.Build()
	rows, err := m.queryMaps(ctx, sql, vars)
	if err != nil {
		return nil, err
	}
	for i, row := range rows {
		if i < len(data) {
			_ = mapToStruct(row, &data[i])
		}
	}
	return data, nil
}

func (m Model[T]) CreateFromMap(ctx context.Context, attrs map[string]any) (*T, error) {
	data := new(T)
	if err := applyMapToStruct(data, m.filterAttrs(attrs)); err != nil {
		return nil, err
	}
	return m.Create(ctx, data)
}

func (m Model[T]) Find(ctx context.Context, id any) (*T, error) {
	if m.state.err != nil {
		return nil, m.state.err
	}
	q := surrealql.Select(m.recordID(id))
	for _, rel := range m.state.with {
		if rel.kind == relFetch {
			q = q.Fetch(rel.jsonName)
		}
	}
	sql, vars := q.Build()
	sql = m.applyTrash(sql)
	rows, err := m.queryAll(ctx, sql, vars)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	if err := m.eagerLoad(ctx, rows); err != nil {
		return nil, err
	}
	return &rows[0], nil
}

func (m Model[T]) FindOrFail(ctx context.Context, id any) (*T, error) {
	v, err := m.Find(ctx, id)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, ErrNotFound
	}
	return v, nil
}

func (m Model[T]) FirstOrCreate(ctx context.Context, attrs map[string]any, values ...map[string]any) (*T, error) {
	v, err := m.whereMap(attrs).First(ctx)
	if err != nil {
		return nil, err
	}
	if v != nil {
		return v, nil
	}
	merged := cloneMap(attrs)
	for _, vals := range values {
		for k, val := range vals {
			merged[k] = val
		}
	}
	return m.CreateFromMap(ctx, merged)
}

func (m Model[T]) FirstOrNew(ctx context.Context, attrs map[string]any, values ...map[string]any) (*T, error) {
	v, err := m.whereMap(attrs).First(ctx)
	if err != nil {
		return nil, err
	}
	if v != nil {
		return v, nil
	}
	merged := cloneMap(attrs)
	for _, vals := range values {
		for k, val := range vals {
			merged[k] = val
		}
	}
	out := new(T)
	if err := applyMapToStruct(out, m.filterAttrs(merged)); err != nil {
		return nil, err
	}
	return out, nil
}

func (m Model[T]) UpdateOrCreate(ctx context.Context, attrs map[string]any, values map[string]any) (*T, error) {
	v, err := m.whereMap(attrs).First(ctx)
	if err != nil {
		return nil, err
	}
	if v != nil {
		id, _ := getID(v)
		if id == nil {
			return nil, ErrNotFound
		}
		return m.Update(ctx, id, values)
	}
	merged := cloneMap(attrs)
	for k, val := range values {
		merged[k] = val
	}
	return m.CreateFromMap(ctx, merged)
}

func (m Model[T]) Save(ctx context.Context, model *T) (*T, error) {
	id, ok := getID(model)
	if !ok {
		return m.Create(ctx, model)
	}
	attrs, err := structToMap(model)
	if err != nil {
		return nil, err
	}
	delete(attrs, "id")
	return m.Update(ctx, id, attrs)
}

func (m Model[T]) Update(ctx context.Context, id any, data map[string]any) (*T, error) {
	model, err := m.Find(ctx, id)
	if err != nil {
		return nil, err
	}
	if model == nil {
		return nil, ErrNotFound
	}
	if err := fireHook(model, hookSaving); err != nil {
		return nil, err
	}
	if err := fireHook(model, hookUpdating); err != nil {
		return nil, err
	}
	updates := cloneMap(data)
	delete(updates, "id")
	applyTimestamps(updates, model, false, m.meta)
	updated, err := m.updateByID(ctx, id, updates)
	if err != nil {
		return nil, err
	}
	fireHook(model, hookUpdated)
	fireHook(model, hookSaved)
	return updated, nil
}

func (m Model[T]) updateByID(ctx context.Context, id any, data map[string]any) (*T, error) {
	rid := m.recordID(id)
	q := surrealql.Update(rid)
	for k, v := range data {
		if k == "id" {
			continue
		}
		setUpdateField(q, k, v)
	}
	q = q.Return(surrealql.ReturnAfterClause)
	sql, vars := q.Build()
	rows, err := m.queryAll(ctx, sql, vars)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	return &rows[0], nil
}

func (m Model[T]) UpdateWhere(ctx context.Context, data map[string]any) ([]T, error) {
	updates := cloneMap(data)
	delete(updates, "id")
	applyTimestamps(updates, nil, false, m.meta)
	q := surrealql.Update(m.table)
	for k, v := range updates {
		if k == "id" {
			continue
		}
		setUpdateField(q, k, v)
	}
	for _, c := range m.state.conds {
		q = q.Where(c.sql, c.args...)
	}
	q = q.Return(surrealql.ReturnAfterClause)
	sql, vars := q.Build()
	return m.queryAll(ctx, sql, vars)
}

// setUpdateField sets a field on an UPDATE query, mapping nil values to the
// SurrealDB NONE keyword so they clear the stored value.
func setUpdateField(q *surrealql.UpdateQuery, k string, v any) {
	if v == nil {
		q.Set(k + " = NONE")
		return
	}
	q.Set(k, v)
}

func (m Model[T]) Delete(ctx context.Context, id any) error {
	model, err := m.Find(ctx, id)
	if err != nil {
		return err
	}
	if model == nil {
		return ErrNotFound
	}
	if err := fireHook(model, hookDeleting); err != nil {
		return err
	}
	if m.meta != nil && m.meta.softDelete {
		now := time.Now()
		q := surrealql.Update(m.recordID(id)).Set("deleted_at", now).Return(surrealql.ReturnAfterClause)
		sql, vars := q.Build()
		if _, err := m.queryMaps(ctx, sql, vars); err != nil {
			return err
		}
		setTimestampField(model, "deleted_at", now)
		fireHook(model, hookDeleted)
		return nil
	}
	q := surrealql.Delete(m.recordID(id))
	sql, vars := q.Build()
	if _, err := m.queryMaps(ctx, sql, vars); err != nil {
		return err
	}
	fireHook(model, hookDeleted)
	return nil
}

func (m Model[T]) DeleteWhere(ctx context.Context) error {
	if m.meta != nil && m.meta.softDelete {
		q := surrealql.Update(m.table).Set("deleted_at", time.Now()).Return(surrealql.ReturnAfterClause)
		for _, c := range m.state.conds {
			q = q.Where(c.sql, c.args...)
		}
		sql, vars := q.Build()
		_, err := m.queryMaps(ctx, sql, vars)
		return err
	}
	q := surrealql.Delete(m.table)
	for _, c := range m.state.conds {
		q = q.Where(c.sql, c.args...)
	}
	sql, vars := q.Build()
	_, err := m.queryMaps(ctx, sql, vars)
	return err
}

func (m Model[T]) ForceDelete(ctx context.Context, id any) error {
	q := surrealql.Delete(m.recordID(id))
	sql, vars := q.Build()
	_, err := m.queryMaps(ctx, sql, vars)
	return err
}

func (m Model[T]) Restore(ctx context.Context, id any) error {
	q := surrealql.Update(m.recordID(id)).Set("deleted_at = NONE").Return(surrealql.ReturnAfterClause)
	sql, vars := q.Build()
	_, err := m.queryMaps(ctx, sql, vars)
	return err
}

func (m Model[T]) Truncate(ctx context.Context) error {
	q := surrealql.Delete(m.table)
	sql, vars := q.Build()
	_, err := m.queryMaps(ctx, sql, vars)
	return err
}
