package surrealgoorm

import (
	"reflect"
	"sync"
)

// modelMetadata stores parsed field info so hot paths do not keep re-reading tags.
type modelMetadata struct {
	columns      []fieldMetadata
	relations    map[string]fieldMetadata
	hasDeletedAt bool
	defaultTable string
}

type fieldMetadata struct {
	IndexPath []int
	Column    string
	Auto      bool
	Relation  relationTag
}

var modelMetadataCache sync.Map

func getModelMetadata(t reflect.Type) *modelMetadata {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if cached, ok := modelMetadataCache.Load(t); ok {
		return cached.(*modelMetadata)
	}

	meta := &modelMetadata{
		relations:    make(map[string]fieldMetadata),
		defaultTable: pluralize(toSnakeCase(t.Name())),
	}
	collectFieldMetadata(t, nil, meta)
	actual, _ := modelMetadataCache.LoadOrStore(t, meta)
	return actual.(*modelMetadata)
}

func collectFieldMetadata(t reflect.Type, path []int, meta *modelMetadata) {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		indexPath := append(append([]int(nil), path...), i)

		if field.Anonymous {
			// Embedded structs are flattened into the parent metadata.
			collectFieldMetadata(field.Type, indexPath, meta)
			continue
		}

		tag := field.Tag.Get("orm")
		if tag == "" {
			continue
		}

		if columnName, ok := columnNameFromTag(tag); ok && columnName != "-" {
			fieldMeta := fieldMetadata{
				IndexPath: indexPath,
				Column:    columnName,
				Auto:      hasORMFlag(tag, "auto"),
			}
			meta.columns = append(meta.columns, fieldMeta)
			if columnName == "deleted_at" {
				meta.hasDeletedAt = true
			}
		}

		if relation, ok := parseRelationMetadata(tag); ok {
			relationField := fieldMetadata{
				IndexPath: indexPath,
				Relation:  relation,
			}
			meta.relations[relation.Name] = relationField
			meta.relations[toSnakeCase(field.Name)] = relationField
		}
	}
}

func valueByIndexPath(v reflect.Value, path []int) reflect.Value {
	for _, index := range path {
		if v.Kind() == reflect.Pointer {
			if v.IsNil() {
				v.Set(reflect.New(v.Type().Elem()))
			}
			v = v.Elem()
		}
		v = v.Field(index)
	}
	return v
}
