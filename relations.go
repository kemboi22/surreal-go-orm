package surrealgoorm

import (
	"context"
	"fmt"

	"github.com/surrealdb/surrealdb.go/contrib/surrealql"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

func (m Model[T]) Relate(ctx context.Context, from models.RecordID, edge string, to models.RecordID, content map[string]any) (map[string]any, error) {
	q := surrealql.Relate(from, edge, to)
	if len(content) > 0 {
		q = q.Content(content)
	}
	sql, vars := q.Build()
	rows, err := m.queryMaps(ctx, sql, vars)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return map[string]any{}, nil
	}
	return rows[0], nil
}

// HasMany returns all records in the related table whose foreign key matches
// the local key of the records produced by this query.
//
// The query should already be narrowed down to the relevant record(s),
// e.g. Query[User](db, "user").WhereEq("id", userID). The local key defaults to
// the "id" field of the parent records.
//
// Example:
//
//	posts, err := surrealgoorm.Query[User](db, "user").
//	    WhereEq("id", userID).
//	    HasMany[Post](ctx, "post", "user_id")
func (m *Model[T]) HasMany[R any](ctx context.Context, relatedTable, foreignKey string) (*[]R, error) {
	parents, err := m.Get(ctx)
	if err != nil {
		return nil, err
	}
	keys := make([]any, 0, len(*parents))
	for i := range *parents {
		if id, ok := getID(&(*parents)[i]); ok {
			keys = append(keys, id)
		}
	}
	if len(keys) == 0 {
		return nil, nil
	}
	q := newModel[R](m.client, relatedTable)
	q.sq.Where(foreignKey+" IN ?", keys)
	return q.Get(ctx)
}

// HasOne is like HasMany but returns a single related record.
func (m *Model[T]) HasOne[R any](ctx context.Context, relatedTable, foreignKey string) (*R, error) {
	items, err := m.HasMany[R](ctx, relatedTable, foreignKey)
	if err != nil {
		return nil, err
	}
	if items == nil || len(*items) == 0 {
		return nil, nil
	}
	return &(*items)[0], nil
}

// BelongsTo returns the parent record that owns the child record produced by
// this query. The child's foreign key is matched against the "id" field
// of the parent table.
//
// Example:
//
//	author, err := surrealgoorm.Query[Post](db, "post").
//	    WhereEq("id", postID).
//	    BelongsTo[User](ctx, "user", "user_id")
func (m *Model[T]) BelongsTo[R any](ctx context.Context, parentTable, foreignKey string) (*R, error) {
	record, err := m.First(ctx)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, nil
	}
	attrs, err := structToMap(record)
	if err != nil {
		return nil, err
	}
	fkValue, ok := attrs[foreignKey]
	if !ok || fkValue == nil {
		return nil, nil
	}
	return newModel[R](m.client, parentTable).Find(ctx, fkValue)
}

// Associate sets the foreign key on the child record to point at the given
// parent record and persists the change.
func (m *Model[T]) Associate[R any](ctx context.Context, foreignKey string, parent *R) (*T, error) {
	record, err := m.First(ctx)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, ErrNotFound
	}
	id, ok := getID(record)
	if !ok || id == nil {
		return nil, ErrNotFound
	}
	parentID, ok := getID(parent)
	if !ok || parentID == nil {
		return nil, ErrNotFound
	}
	return m.Update(ctx, id, map[string]any{foreignKey: parentID})
}

// Dissociate clears the foreign key on the child record and persists the change.
func (m *Model[T]) Dissociate(ctx context.Context, foreignKey string) (*T, error) {
	record, err := m.First(ctx)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, ErrNotFound
	}
	id, ok := getID(record)
	if !ok || id == nil {
		return nil, ErrNotFound
	}
	return m.Update(ctx, id, map[string]any{foreignKey: nil})
}

// HasMany returns all records in the related table whose foreign key matches
// the local key of the records produced by the parent query.
//
// Deprecated: use Query[T](...).HasMany[R](...) instead.
func HasMany[T any, R any](ctx context.Context, parent QueryBuilder[T], relatedTable, foreignKey string) (*[]R, error) {
	m, ok := parent.(*Model[T])
	if !ok {
		return nil, fmt.Errorf("surrealgoorm: query builder does not support relationships")
	}
	return m.HasMany[R](ctx, relatedTable, foreignKey)
}

// HasOne is like HasMany but returns a single related record.
//
// Deprecated: use Query[T](...).HasOne[R](...) instead.
func HasOne[T any, R any](ctx context.Context, parent QueryBuilder[T], relatedTable, foreignKey string) (*R, error) {
	m, ok := parent.(*Model[T])
	if !ok {
		return nil, fmt.Errorf("surrealgoorm: query builder does not support relationships")
	}
	return m.HasOne[R](ctx, relatedTable, foreignKey)
}

// BelongsTo returns the parent record that owns the child record produced by
// the child query.
//
// Deprecated: use Query[T](...).BelongsTo[R](...) instead.
func BelongsTo[T any, R any](ctx context.Context, child QueryBuilder[T], parentTable, foreignKey string) (*R, error) {
	m, ok := child.(*Model[T])
	if !ok {
		return nil, fmt.Errorf("surrealgoorm: query builder does not support relationships")
	}
	return m.BelongsTo[R](ctx, parentTable, foreignKey)
}

// Associate sets the foreign key on the child record to point at the given
// parent record and persists the change.
//
// Deprecated: use Query[T](...).Associate[R](...) instead.
func Associate[T any, R any](ctx context.Context, child QueryBuilder[T], foreignKey string, parent *R) (*T, error) {
	m, ok := child.(*Model[T])
	if !ok {
		return nil, fmt.Errorf("surrealgoorm: query builder does not support relationships")
	}
	return m.Associate[R](ctx, foreignKey, parent)
}

// Dissociate clears the foreign key on the child record and persists the change.
//
// Deprecated: use Query[T](...).Dissociate(...) instead.
func Dissociate[T any](ctx context.Context, child QueryBuilder[T], foreignKey string) (*T, error) {
	m, ok := child.(*Model[T])
	if !ok {
		return nil, fmt.Errorf("surrealgoorm: query builder does not support relationships")
	}
	return m.Dissociate(ctx, foreignKey)
}

func clientOf[T any](q QueryBuilder[T]) (any, error) {
	cp, ok := q.(connProvider)
	if !ok {
		return nil, fmt.Errorf("surrealgoorm: query builder does not support relationships")
	}
	return cp.conn(), nil
}
