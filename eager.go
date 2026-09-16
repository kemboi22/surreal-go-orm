package surrealgoorm

import (
	"context"
	"fmt"
	"reflect"

	"github.com/surrealdb/surrealdb.go/contrib/surrealql"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

func (m Model[T]) eagerLoad(ctx context.Context, parents []T) error {
	if len(parents) == 0 || len(m.state.with) == 0 {
		return nil
	}
	for _, rel := range m.state.with {
		switch rel.kind {
		case relFetch:
			// SurrealDB FETCH already expanded nested objects into the parent rows.
			continue
		case relHasMany, relHasOne:
			if err := m.eagerHasMany(ctx, parents, rel); err != nil {
				return err
			}
		case relBelongsTo:
			if err := m.eagerBelongsTo(ctx, parents, rel); err != nil {
				return err
			}
		default:
			return fmt.Errorf("surrealgoorm: unsupported relation kind for %q", rel.fieldName)
		}
	}
	return nil
}

func (m Model[T]) eagerHasMany(ctx context.Context, parents []T, rel *relationMeta) error {
	keys := make([]any, 0, len(parents))
	seen := map[string]bool{}
	for i := range parents {
		id, ok := getID(&parents[i])
		if !ok || id == nil {
			continue
		}
		key := idKey(id)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		keys = append(keys, id)
	}
	if len(keys) == 0 {
		return nil
	}

	records, err := m.queryRelated(ctx, rel.table, rel.fk+" IN ?", keys)
	if err != nil {
		return err
	}

	grouped := map[string][]reflect.Value{}
	for _, rec := range records {
		item, err := mapToRelated(rel.elemType, rec)
		if err != nil {
			return err
		}
		fkVal, ok := rec[rel.fk]
		if !ok || fkVal == nil {
			continue
		}
		key := idKey(fkVal)
		grouped[key] = append(grouped[key], item)
	}

	for i := range parents {
		id, ok := getID(&parents[i])
		if !ok || id == nil {
			continue
		}
		items := grouped[idKey(id)]
		if err := setRelationField(&parents[i], rel, items); err != nil {
			return err
		}
	}
	return nil
}

func (m Model[T]) eagerBelongsTo(ctx context.Context, parents []T, rel *relationMeta) error {
	keys := make([]any, 0, len(parents))
	seen := map[string]bool{}
	for i := range parents {
		fkVal, ok := fieldJSONValue(&parents[i], rel.fk)
		if !ok || fkVal == nil {
			continue
		}
		key := idKey(fkVal)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		keys = append(keys, fkVal)
	}
	if len(keys) == 0 {
		return nil
	}

	records, err := m.queryRelated(ctx, rel.table, "id IN ?", keys)
	if err != nil {
		return err
	}

	byID := map[string]reflect.Value{}
	for _, rec := range records {
		item, err := mapToRelated(rel.elemType, rec)
		if err != nil {
			return err
		}
		idVal, ok := rec["id"]
		if !ok {
			continue
		}
		byID[idKey(idVal)] = item
	}

	for i := range parents {
		fkVal, ok := fieldJSONValue(&parents[i], rel.fk)
		if !ok || fkVal == nil {
			continue
		}
		item, ok := byID[idKey(fkVal)]
		if !ok {
			continue
		}
		if err := setRelationField(&parents[i], rel, []reflect.Value{item}); err != nil {
			return err
		}
	}
	return nil
}

func (m Model[T]) queryRelated(ctx context.Context, table, cond string, keys []any) ([]map[string]any, error) {
	q := surrealql.Select(table).Where(cond, keys)
	sql, vars := q.Build()
	return m.queryMaps(ctx, sql, vars)
}

func mapToRelated(elemType reflect.Type, rec map[string]any) (reflect.Value, error) {
	ptr := reflect.New(elemType)
	if err := mapToStruct(rec, ptr.Interface()); err != nil {
		return reflect.Value{}, err
	}
	return ptr.Elem(), nil
}

func setRelationField(parent any, rel *relationMeta, items []reflect.Value) error {
	rv := reflect.ValueOf(parent)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return fmt.Errorf("surrealgoorm: nil parent when setting relation %q", rel.fieldName)
		}
		rv = rv.Elem()
	}
	if rel.index < 0 || rel.index >= rv.NumField() {
		return fmt.Errorf("surrealgoorm: invalid relation field index for %q", rel.fieldName)
	}
	fv := rv.Field(rel.index)
	if !fv.CanSet() {
		return fmt.Errorf("surrealgoorm: cannot set relation field %q", rel.fieldName)
	}

	switch rel.kind {
	case relHasMany:
		slice := reflect.MakeSlice(rel.fieldType, 0, len(items))
		for _, item := range items {
			slice = reflect.Append(slice, coerceRelatedValue(rel.fieldType.Elem(), item))
		}
		fv.Set(slice)
	case relHasOne, relBelongsTo:
		if len(items) == 0 {
			fv.Set(reflect.Zero(rel.fieldType))
			return nil
		}
		fv.Set(coerceRelatedValue(rel.fieldType, items[0]))
	}
	return nil
}

func coerceRelatedValue(dstType reflect.Type, item reflect.Value) reflect.Value {
	if dstType.Kind() == reflect.Pointer {
		if item.Kind() == reflect.Pointer {
			return item
		}
		ptr := reflect.New(dstType.Elem())
		ptr.Elem().Set(item)
		return ptr
	}
	if item.Kind() == reflect.Pointer {
		return item.Elem()
	}
	return item
}

func fieldJSONValue(v any, jsonName string) (any, bool) {
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil, false
		}
		rv = rv.Elem()
	}
	var found any
	ok := false
	walkFields(rv, func(name string, fv reflect.Value) {
		if name != jsonName || ok {
			return
		}
		fv = indirect(fv)
		if fv.IsValid() && !fv.IsZero() {
			found = fv.Interface()
			ok = true
		}
	})
	return found, ok
}

func idKey(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case models.RecordID:
		return t.String()
	case *models.RecordID:
		if t == nil {
			return ""
		}
		return t.String()
	default:
		return fmt.Sprintf("%v", t)
	}
}
