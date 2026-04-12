package surrealgoorm

import (
	"context"
	"reflect"
	"strings"

	"github.com/surrealdb/surrealdb.go/pkg/models"
)

func (db *DB) With(ctx context.Context, model any, relations ...string) error {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	if v.Kind() == reflect.Slice {
		slice := v
		for i := 0; i < slice.Len(); i++ {
			elem := slice.Index(i).Addr().Interface()
			if err := loadSingleModelRelations(ctx, db, elem, relations); err != nil {
				return err
			}
		}
		return nil
	}

	return loadSingleModelRelations(ctx, db, model, relations)
}

func loadSingleModelRelations(ctx context.Context, db *DB, model any, relations []string) error {
	for _, rel := range relations {
		if err := loadRelationPath(ctx, db, model, rel); err != nil {
			return err
		}
	}
	return nil
}

func loadRelationPath(ctx context.Context, db *DB, model any, relationPath string) error {
	head, tail, _ := strings.Cut(relationPath, ".")
	// Load the current relation first, then recurse into the remainder.
	if err := loadRelation(ctx, db, model, head); err != nil {
		return err
	}
	if tail == "" {
		return nil
	}

	relatedValue, ok := getRelationValue(model, head)
	if !ok || !relatedValue.IsValid() {
		return nil
	}

	if relatedValue.Kind() == reflect.Pointer && relatedValue.IsNil() {
		return nil
	}

	return db.With(ctx, relatedValue.Interface(), tail)
}

func loadRelation(ctx context.Context, db *DB, model any, relation string) error {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil
	}

	t := v.Type()
	meta := getModelMetadata(t)
	fieldMeta, ok := meta.relations[relation]
	if !ok {
		return nil
	}
	field := valueByIndexPath(v, fieldMeta.IndexPath)
	fieldType := t.FieldByIndex(fieldMeta.IndexPath)

	switch fieldMeta.Relation.Type {
	case "has_many":
		return loadHasMany(ctx, db, model, field, fieldMeta.Relation, fieldType)
	case "has_one":
		return loadHasOne(ctx, db, model, field, fieldMeta.Relation, fieldType)
	case "belongs_to":
		return loadBelongsTo(ctx, db, model, field, fieldMeta.Relation, fieldType)
	default:
		return nil
	}
}

func loadHasMany(ctx context.Context, db *DB, model any, field reflect.Value, meta relationTag, fieldType reflect.StructField) error {
	recordID := getModelRecordIDString(model)
	if recordID == "" {
		return nil
	}

	foreignKey := meta.ForeignKey
	if foreignKey == "" {
		foreignKey = defaultForeignKeyForModel(model)
	}

	elemType := fieldType.Type.Elem()
	slice := reflect.MakeSlice(field.Type(), 0, 0)

	sql := "SELECT * FROM " + meta.Name + " WHERE " + foreignKey + " = $id"
	resp, err := db.query(ctx, sql, map[string]any{"id": recordID})
	if err != nil {
		return err
	}

	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil {
		field.Set(slice)
		return nil
	}

	for _, row := range (*resp)[0].Result {
		elem := reflect.New(elemType)
		mapToModelWithRelations(elem.Interface(), row)
		slice = reflect.Append(slice, elem.Elem())
	}

	field.Set(slice)
	return nil
}

func loadHasOne(ctx context.Context, db *DB, model any, field reflect.Value, meta relationTag, fieldType reflect.StructField) error {
	recordID := getModelRecordIDString(model)
	if recordID == "" {
		return nil
	}

	foreignKey := meta.ForeignKey
	if foreignKey == "" {
		foreignKey = defaultForeignKeyForModel(model)
	}

	sql := "SELECT * FROM " + meta.Name + " WHERE " + foreignKey + " = $id LIMIT 1"
	resp, err := db.query(ctx, sql, map[string]any{"id": recordID})
	if err != nil {
		return err
	}

	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil || len((*resp)[0].Result) == 0 {
		return nil
	}

	elemType := fieldType.Type.Elem()
	elem := reflect.New(elemType)
	mapToModelWithRelations(elem.Interface(), (*resp)[0].Result[0])
	if field.Kind() == reflect.Pointer {
		field.Set(elem)
		return nil
	}
	field.Set(elem.Elem())

	return nil
}

func loadBelongsTo(ctx context.Context, db *DB, model any, field reflect.Value, meta relationTag, fieldType reflect.StructField) error {
	foreignField := meta.ForeignKey
	if foreignField == "" {
		foreignField = fieldType.Name + "ID"
	}

	foreignValue := getFieldValue(model, foreignField)
	if foreignValue == nil {
		return nil
	}

	var rid models.RecordID
	switch v := foreignValue.(type) {
	case models.RecordID:
		rid = v
	case string:
		if strings.Contains(v, ":") {
			parts := strings.SplitN(v, ":", 2)
			rid = models.NewRecordID(parts[0], parts[1])
		} else {
			rid = models.NewRecordID(meta.Name, v)
		}
	default:
		return nil
	}

	elemType := fieldType.Type.Elem()
	if elemType.Kind() == reflect.Pointer {
		elemType = elemType.Elem()
	}

	result := reflect.New(elemType)
	record, err := selectRecord[map[string]any](ctx, db, rid)
	if err != nil {
		return err
	}
	if record == nil {
		return ErrNotFound
	}

	if err := mapToStruct(*record, result.Interface()); err != nil {
		return err
	}

	if field.Kind() == reflect.Pointer {
		field.Set(result)
		return nil
	}
	field.Set(result.Elem())
	return nil
}

func getFieldValue(model any, field string) any {
	if model == nil {
		return nil
	}

	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil
	}

	for i := 0; i < v.NumField(); i++ {
		if v.Type().Field(i).Name == field {
			return v.Field(i).Interface()
		}
	}

	return nil
}

func getRelationValue(model any, relation string) (reflect.Value, bool) {
	if model == nil {
		return reflect.Value{}, false
	}

	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return reflect.Value{}, false
	}

	meta := getModelMetadata(v.Type())
	fieldMeta, ok := meta.relations[relation]
	if !ok {
		return reflect.Value{}, false
	}

	return valueByIndexPath(v, fieldMeta.IndexPath), true
}

func mapToModelWithRelations(model any, row map[string]any) {
	_ = mapToStruct(row, model)
}
