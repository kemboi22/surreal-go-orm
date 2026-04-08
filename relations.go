package surrealgoorm

import (
	"context"
	"reflect"

	"github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

func (db *DB) With(ctx context.Context, model interface{}, relations ...string) error {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
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

func loadSingleModelRelations(ctx context.Context, db *DB, model interface{}, relations []string) error {
	for _, rel := range relations {
		if err := loadRelation(ctx, db, model, rel); err != nil {
			return err
		}
	}
	return nil
}

func loadRelation(ctx context.Context, db *DB, model interface{}, relation string) error {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		tag := fieldType.Tag.Get("orm")

		relType, relName := parseRelationTag(tag)
		if relName != relation {
			continue
		}

		switch relType {
		case "has_many":
			return loadHasMany(ctx, db, model, field, relation, fieldType)
		case "has_one":
			return loadHasOne(ctx, db, model, field, relation, fieldType)
		case "belongs_to":
			return loadBelongsTo(ctx, db, model, field, relation, fieldType)
		}

		break
	}

	return nil
}

func loadHasMany(ctx context.Context, db *DB, model interface{}, field reflect.Value, relation string, fieldType reflect.StructField) error {
	modelID := getModelID(model)
	if modelID == "" {
		return nil
	}

	foreignKey := getForeignKeyFromTag(fieldType.Tag, "user_id")

	elemType := fieldType.Type.Elem()
	sliceType := reflect.SliceOf(elemType)
	slice := reflect.New(sliceType)

	sql := "SELECT * FROM " + relation + " WHERE " + foreignKey + " = ?"
	resp, err := surrealdb.Query[[]map[string]any](ctx, db.raw, sql, map[string]any{"0": "users:" + modelID})
	if err != nil {
		return err
	}

	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil {
		field.Set(slice.Elem())
		return nil
	}

	for _, row := range (*resp)[0].Result {
		elem := reflect.New(elemType)
		mapToModelWithRelations(elem.Interface(), row)
		slice = reflect.Append(slice, elem.Elem())
	}

	field.Set(slice.Elem())
	return nil
}

func loadHasOne(ctx context.Context, db *DB, model interface{}, field reflect.Value, relation string, fieldType reflect.StructField) error {
	modelID := getModelID(model)
	if modelID == "" {
		return nil
	}

	foreignKey := getForeignKeyFromTag(fieldType.Tag, "user_id")

	sql := "SELECT * FROM " + relation + " WHERE " + foreignKey + " = ? LIMIT 1"
	resp, err := surrealdb.Query[[]map[string]any](ctx, db.raw, sql, map[string]any{"0": "users:" + modelID})
	if err != nil {
		return err
	}

	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil || len((*resp)[0].Result) == 0 {
		return nil
	}

	elemType := fieldType.Type.Elem()
	elem := reflect.New(elemType)
	mapToModelWithRelations(elem.Interface(), (*resp)[0].Result[0])
	field.Set(elem)

	return nil
}

func loadBelongsTo(ctx context.Context, db *DB, model interface{}, field reflect.Value, relation string, fieldType reflect.StructField) error {
	foreignValue := getFieldValue(model, fieldType.Name)
	if foreignValue == nil {
		return nil
	}

	var rid models.RecordID
	switch v := foreignValue.(type) {
	case models.RecordID:
		rid = v
	case string:
		rid = models.NewRecordID(relation, v)
	default:
		return nil
	}

	elemType := fieldType.Type.Elem()
	if elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}

	result := reflect.New(elemType)
	_, err := surrealdb.Select[map[string]any](ctx, db.raw, rid)
	if err != nil {
		return err
	}

	field.Set(result)
	return nil
}

func parseRelationTag(tag string) (relationType, name string) {
	parts := splitTag(tag)
	if len(parts) >= 2 {
		return parts[1], parts[0]
	}
	return "", ""
}

func getForeignKeyFromTag(tag reflect.StructTag, fallback string) string {
	if val := tag.Get("foreign_key"); val != "" {
		return val
	}
	return fallback
}

func getModelID(model interface{}) string {
	if model == nil {
		return ""
	}

	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return ""
	}

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if field.Type() == reflect.TypeOf(models.RecordID{}) {
			if rid, ok := field.Interface().(models.RecordID); ok {
				return rid.ID.(string)
			}
		}
		if field.Type() == reflect.TypeOf("") {
			if fieldName := v.Type().Field(i).Name; fieldName == "ID" {
				if id, ok := field.Interface().(string); ok {
					return id
				}
			}
		}
	}

	return ""
}

func getFieldValue(model interface{}, field string) interface{} {
	if model == nil {
		return nil
	}

	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
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

func mapToModelWithRelations(model interface{}, row map[string]any) {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		tag := fieldType.Tag.Get("orm")

		if tag == "" {
			continue
		}

		parts := splitTag(tag)
		columnName := parts[0]

		if columnName == "" || columnName == "-" {
			continue
		}

		if val, ok := row[columnName]; ok && val != nil && field.CanSet() {
			field.Set(reflect.ValueOf(val))
		}
	}
}
