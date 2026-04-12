package surrealgoorm

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"unicode"

	"github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

type relationTag struct {
	Type       string
	Name       string
	ForeignKey string
}

func parseORMTagParts(tag string) []string {
	if tag == "" {
		return nil
	}

	rawParts := strings.Split(tag, ";")
	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		part = strings.TrimSpace(part)
		if part != "" {
			parts = append(parts, part)
		}
	}

	return parts
}

func tagKeyValue(part string) (string, string) {
	key, value, ok := strings.Cut(part, ":")
	if !ok {
		return strings.TrimSpace(part), ""
	}
	return strings.TrimSpace(key), strings.TrimSpace(value)
}

func columnNameFromTag(tag string) (string, bool) {
	for _, part := range parseORMTagParts(tag) {
		key, value := tagKeyValue(part)
		if key == "column" && value != "" {
			return value, true
		}
	}
	return "", false
}

func hasORMFlag(tag string, flag string) bool {
	for _, part := range parseORMTagParts(tag) {
		if part == flag {
			return true
		}
	}
	return false
}

func parseRelationMetadata(tag string) (relationTag, bool) {
	meta := relationTag{}
	for _, part := range parseORMTagParts(tag) {
		key, value := tagKeyValue(part)
		switch key {
		case "has_many", "has_one", "belongs_to":
			meta.Type = key
			meta.Name = value
		case "foreign_key":
			meta.ForeignKey = value
		}
	}

	return meta, meta.Type != "" && meta.Name != ""
}

func buildConditionSQL(w WhereCondition, index int, stringifyID bool) string {
	if w.Operator == "" {
		return formatRawCondition(w.Field, index, rawBindings(w.Value))
	}

	field := w.Field
	if stringifyID && w.Field == "id" && w.Operator == "=" {
		field = "string::concat(id)"
	}

	switch w.Operator {
	case "BETWEEN":
		return fmt.Sprintf("%s BETWEEN $w%d[0] AND $w%d[1]", field, index, index)
	case "IS":
		return field + " IS NONE"
	case "IS NOT":
		return field + " IS NOT NONE"
	default:
		return fmt.Sprintf("%s %s $w%d", field, w.Operator, index)
	}
}

func rawBindings(v any) []any {
	bindings, ok := v.([]any)
	if !ok {
		return nil
	}
	return bindings
}

func flattenBindings(wheres []WhereCondition) []any {
	var bindings []any
	for _, where := range wheres {
		if where.Operator == "" {
			bindings = append(bindings, rawBindings(where.Value)...)
			continue
		}
		if where.Value != nil {
			bindings = append(bindings, where.Value)
		}
	}
	return bindings
}

func formatRawCondition(sql string, index int, bindings []any) string {
	condition := sql
	for bindIndex := range bindings {
		placeholder := fmt.Sprintf("$w%d_%d", index, bindIndex)
		condition = strings.Replace(condition, "?", placeholder, 1)
	}
	return condition
}

func buildWhereClause(wheres []WhereCondition, stringifyID bool) string {
	if len(wheres) == 0 {
		return ""
	}

	var sql strings.Builder
	for i, w := range wheres {
		if i > 0 {
			if w.Or {
				sql.WriteString(" OR ")
			} else {
				sql.WriteString(" AND ")
			}
		}
		sql.WriteString(buildConditionSQL(w, i, stringifyID))
	}

	return sql.String()
}

func buildWhereParams(wheres []WhereCondition) map[string]any {
	params := make(map[string]any)
	for i, w := range wheres {
		switch w.Operator {
		case "":
			for bindIndex, binding := range rawBindings(w.Value) {
				params[fmt.Sprintf("w%d_%d", i, bindIndex)] = binding
			}
		case "IS", "IS NOT":
			continue
		default:
			if w.Value != nil {
				params[fmt.Sprintf("w%d", i)] = w.Value
			}
		}
	}
	return params
}

func appendSoftDeleteClause(sql *strings.Builder, hasWhere bool, enabled bool, withTrashed bool, onlyTrashed bool) {
	if !enabled || withTrashed {
		return
	}

	if hasWhere {
		sql.WriteString(" AND ")
	} else {
		sql.WriteString(" WHERE ")
	}

	if onlyTrashed {
		sql.WriteString("deleted_at IS NOT NONE")
		return
	}

	sql.WriteString("deleted_at IS NONE")
}

func ensureTableName(name string) error {
	if strings.TrimSpace(name) == "" {
		return ErrEmptyTableName
	}
	return nil
}

func ensureSafeMutation(allowAll bool, wheres []WhereCondition) error {
	if allowAll || len(wheres) > 0 {
		return nil
	}
	return fmt.Errorf("%w: %w", ErrUnsafeMutation, ErrMissingWhereClause)
}

func buildUpdateSetClause(data map[string]any) (string, map[string]any) {
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	params := make(map[string]any, len(keys))
	var sql strings.Builder
	for i, key := range keys {
		if i > 0 {
			sql.WriteString(", ")
		}
		paramName := fmt.Sprintf("u%d", i)
		fmt.Fprintf(&sql, "%s = $%s", key, paramName)
		params[paramName] = data[key]
	}

	return sql.String(), params
}

func mergeParams(base map[string]any, extra map[string]any) map[string]any {
	if len(base) == 0 && len(extra) == 0 {
		return nil
	}

	out := make(map[string]any, len(base)+len(extra))
	for key, value := range base {
		out[key] = value
	}
	for key, value := range extra {
		out[key] = value
	}
	return out
}

func modelHasDeletedAt(t reflect.Type) bool {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return false
	}
	return getModelMetadata(t).hasDeletedAt
}

func modelSupportsSoftDelete[T any]() bool {
	return modelHasDeletedAt(reflect.TypeFor[T]())
}

func getModelRecordID(model any) (models.RecordID, bool) {
	if model == nil {
		return models.RecordID{}, false
	}

	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return models.RecordID{}, false
	}

	meta := getModelMetadata(v.Type())
	for _, fieldMeta := range meta.columns {
		if fieldMeta.Column != "id" {
			continue
		}
		field := valueByIndexPath(v, fieldMeta.IndexPath)
		if field.Type() == reflect.TypeFor[models.RecordID]() {
			rid, ok := field.Interface().(models.RecordID)
			if ok && rid.String() != "" {
				return rid, true
			}
		}
	}

	return models.RecordID{}, false
}

func getModelRecordIDString(model any) string {
	if rid, ok := getModelRecordID(model); ok {
		return rid.String()
	}

	if model == nil {
		return ""
	}

	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return ""
	}

	field := v.FieldByName("ID")
	if field.IsValid() && field.Kind() == reflect.String {
		return field.String()
	}

	return ""
}

func defaultForeignKeyForModel(model any) string {
	table := getTableNameFromValue(model)
	if table == "" {
		return "parent_id"
	}
	return singularize(table) + "_id"
}

func getTableNameFromValue(model any) string {
	if model == nil {
		return ""
	}

	if tn, ok := model.(TableName); ok && tn.TableName() != "" {
		return tn.TableName()
	}

	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return ""
	}

	return getModelMetadata(t).defaultTable
}

func singularize(name string) string {
	switch {
	case strings.HasSuffix(name, "ies") && len(name) > 3:
		return name[:len(name)-3] + "y"
	case strings.HasSuffix(name, "s") && len(name) > 1:
		return name[:len(name)-1]
	default:
		return name
	}
}

func pluralize(name string) string {
	switch {
	case strings.HasSuffix(name, "y") && len(name) > 1:
		prev := rune(name[len(name)-2])
		if !strings.ContainsRune("aeiou", unicode.ToLower(prev)) {
			return name[:len(name)-1] + "ies"
		}
	case strings.HasSuffix(name, "s"), strings.HasSuffix(name, "x"), strings.HasSuffix(name, "z"),
		strings.HasSuffix(name, "ch"), strings.HasSuffix(name, "sh"):
		return name + "es"
	}

	return name + "s"
}

func toSnakeCase(s string) string {
	if s == "" {
		return ""
	}

	var out strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				prev := rune(s[i-1])
				var next rune
				if i+1 < len(s) {
					next = rune(s[i+1])
				}
				if unicode.IsLower(prev) || (next != 0 && unicode.IsLower(next)) {
					out.WriteByte('_')
				}
			}
			out.WriteRune(unicode.ToLower(r))
			continue
		}
		out.WriteRune(r)
	}

	return out.String()
}

func (db *DB) query(ctx context.Context, sql string, params map[string]any) (*[]surrealdb.QueryResult[[]map[string]any], error) {
	if db.queryOverride != nil {
		return db.queryOverride(ctx, sql, params)
	}
	if db.tx != nil {
		return surrealdb.Query[[]map[string]any](ctx, db.tx, sql, params)
	}
	return surrealdb.Query[[]map[string]any](ctx, db.raw, sql, params)
}

func (db *DB) execQuery(ctx context.Context, sql string, params map[string]any) (*[]surrealdb.QueryResult[any], error) {
	if db.execOverride != nil {
		return db.execOverride(ctx, sql, params)
	}
	if db.tx != nil {
		return surrealdb.Query[any](ctx, db.tx, sql, params)
	}
	return surrealdb.Query[any](ctx, db.raw, sql, params)
}

func selectRecord[T any, TWhat surrealdb.TableOrRecord](ctx context.Context, db *DB, what TWhat) (*T, error) {
	if db.selectOverride != nil {
		value, err := db.selectOverride(ctx, what)
		if err != nil || value == nil {
			return nil, err
		}
		if typed, ok := value.(T); ok {
			return &typed, nil
		}
		if typed, ok := value.(*T); ok {
			return typed, nil
		}
		return nil, fmt.Errorf("select override returned wrong type")
	}
	if db.tx != nil {
		return surrealdb.Select[T](ctx, db.tx, what)
	}
	return surrealdb.Select[T](ctx, db.raw, what)
}

func createRecord[T any, TWhat surrealdb.TableOrRecord](ctx context.Context, db *DB, what TWhat, data any) (*T, error) {
	if db.createOverride != nil {
		value, err := db.createOverride(ctx, what, data)
		if err != nil || value == nil {
			return nil, err
		}
		if typed, ok := value.(T); ok {
			return &typed, nil
		}
		if typed, ok := value.(*T); ok {
			return typed, nil
		}
		return nil, fmt.Errorf("create override returned wrong type")
	}
	if db.tx != nil {
		return surrealdb.Create[T](ctx, db.tx, what, data)
	}
	return surrealdb.Create[T](ctx, db.raw, what, data)
}

func updateRecord[T any, TWhat surrealdb.TableOrRecord](ctx context.Context, db *DB, what TWhat, data any) (*T, error) {
	if db.updateOverride != nil {
		value, err := db.updateOverride(ctx, what, data)
		if err != nil || value == nil {
			return nil, err
		}
		if typed, ok := value.(T); ok {
			return &typed, nil
		}
		if typed, ok := value.(*T); ok {
			return typed, nil
		}
		return nil, fmt.Errorf("update override returned wrong type")
	}
	if db.tx != nil {
		return surrealdb.Update[T](ctx, db.tx, what, data)
	}
	return surrealdb.Update[T](ctx, db.raw, what, data)
}

func deleteRecord[T any, TWhat surrealdb.TableOrRecord](ctx context.Context, db *DB, what TWhat) (*T, error) {
	if db.deleteOverride != nil {
		value, err := db.deleteOverride(ctx, what)
		if err != nil || value == nil {
			return nil, err
		}
		if typed, ok := value.(T); ok {
			return &typed, nil
		}
		if typed, ok := value.(*T); ok {
			return typed, nil
		}
		return nil, fmt.Errorf("delete override returned wrong type")
	}
	if db.tx != nil {
		return surrealdb.Delete[T](ctx, db.tx, what)
	}
	return surrealdb.Delete[T](ctx, db.raw, what)
}

func insertRelation[T any](ctx context.Context, db *DB, relationship *surrealdb.Relationship) (*T, error) {
	if db.tx != nil {
		return surrealdb.InsertRelation[T](ctx, db.tx, relationship)
	}
	return surrealdb.InsertRelation[T](ctx, db.raw, relationship)
}
