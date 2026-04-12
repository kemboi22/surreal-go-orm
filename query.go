package surrealgoorm

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/surrealdb/surrealdb.go/pkg/models"
)

type WhereCondition struct {
	Field    string
	Operator string
	Value    any
	Or       bool
}

type OrderBy struct {
	Field string
	Desc  bool
}

func Raw(sql string, bindings ...any) *RawQuery {
	return &RawQuery{
		sql:      sql,
		bindings: bindings,
	}
}

type RawQuery struct {
	sql      string
	bindings []any
}

func (rq *RawQuery) String() string {
	return rq.sql
}

func (rq *RawQuery) Bindings() []any {
	return rq.bindings
}

func (db *DB) RawQuery(sql string, bindings ...any) *RawBuilder {
	return &RawBuilder{
		db:       db,
		sql:      sql,
		bindings: bindings,
	}
}

type RawBuilder struct {
	db       *DB
	sql      string
	bindings []any
}

func (rb *RawBuilder) SQL() string {
	return rb.sql
}

func (rb *RawBuilder) Bindings() []any {
	return rb.bindings
}

func (rb *RawBuilder) Exec(ctx context.Context) error {
	_, err := rb.db.execQuery(ctx, rb.sql, rb.buildParams())
	return err
}

func (rb *RawBuilder) All(ctx context.Context, results any) error {
	resp, err := rb.db.query(ctx, rb.sql, rb.buildParams())
	if err != nil {
		return err
	}

	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil {
		return nil
	}

	return mapSliceToStruct((*resp)[0].Result, results)
}

func (rb *RawBuilder) One(ctx context.Context, result any) error {
	rb.sql = addLimitOne(rb.sql)

	resp, err := rb.db.query(ctx, rb.sql, rb.buildParams())
	if err != nil {
		return err
	}

	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil || len((*resp)[0].Result) == 0 {
		return ErrNotFound
	}

	return mapToStruct((*resp)[0].Result[0], result)
}

func (rb *RawBuilder) Scalar(ctx context.Context) (any, error) {
	resp, err := rb.db.query(ctx, rb.sql, rb.buildParams())
	if err != nil {
		return nil, err
	}

	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil || len((*resp)[0].Result) == 0 {
		return nil, nil
	}

	return (*resp)[0].Result[0], nil
}

func (rb *RawBuilder) buildParams() map[string]any {
	params := make(map[string]any)
	for i, binding := range rb.bindings {
		params[fmt.Sprintf("p%d", i)] = binding
	}
	return params
}

func addLimitOne(sql string) string {
	if !strings.HasSuffix(strings.TrimSpace(sql), "LIMIT 1") {
		return sql + " LIMIT 1"
	}
	return sql
}

func mapSliceToStruct(rows []map[string]any, results any) error {
	v := reflect.ValueOf(results)
	if v.Kind() != reflect.Pointer {
		return fmt.Errorf("results must be a pointer")
	}

	slice := v.Elem()
	if !slice.IsValid() {
		return fmt.Errorf("invalid slice")
	}

	if slice.Type().Elem() == reflect.TypeOf(map[string]any{}) {
		for _, row := range rows {
			slice.Set(reflect.Append(slice, reflect.ValueOf(row)))
		}
		return nil
	}

	elemType := slice.Type().Elem()
	for _, row := range rows {
		elem := reflect.New(elemType)
		if err := mapToStruct(row, elem.Interface()); err != nil {
			continue
		}
		slice.Set(reflect.Append(slice, elem.Elem()))
	}

	return nil
}

func (db *DB) Table(table string) *TableQuery {
	return &TableQuery{
		db:    db,
		table: table,
	}
}

type TableQuery struct {
	db         *DB
	table      string
	selects    []string
	selectRaws []string
	wheres     []WhereCondition
	orders     []OrderBy
	orderRaws  []string
	limitVal   int
	offsetVal  int
	joins      []JoinCondition
	allowAll   bool
}

type JoinCondition struct {
	Type    string
	Table   string
	OnLeft  string
	OnRight string
	OnOp    string
}

func (tq *TableQuery) Select(fields ...string) *TableQuery {
	tq.selects = fields
	return tq
}

func (tq *TableQuery) SelectRaw(sql string) *TableQuery {
	tq.selectRaws = append(tq.selectRaws, sql)
	return tq
}

func (tq *TableQuery) Where(field string, op string, value any) *TableQuery {
	tq.wheres = append(tq.wheres, WhereCondition{
		Field:    field,
		Operator: op,
		Value:    value,
	})
	return tq
}

func (tq *TableQuery) WhereRaw(sql string, bindings ...any) *TableQuery {
	tq.wheres = append(tq.wheres, WhereCondition{
		Field:    "(" + sql + ")",
		Operator: "",
		Value:    bindings,
	})
	return tq
}

func (tq *TableQuery) WhereIn(field string, values []any) *TableQuery {
	tq.wheres = append(tq.wheres, WhereCondition{
		Field:    field,
		Operator: "IN",
		Value:    values,
	})
	return tq
}

func (tq *TableQuery) WhereBetween(field string, start, end any) *TableQuery {
	tq.wheres = append(tq.wheres, WhereCondition{
		Field:    field,
		Operator: "BETWEEN",
		Value:    []any{start, end},
	})
	return tq
}

func (tq *TableQuery) WhereLike(field string, value string) *TableQuery {
	tq.wheres = append(tq.wheres, WhereCondition{Field: field, Operator: "CONTAINS", Value: value})
	return tq
}

func (tq *TableQuery) WhereExists(subquery string, bindings ...any) *TableQuery {
	tq.wheres = append(tq.wheres, WhereCondition{
		Field: "(" + "EXISTS(" + formatRawCondition(subquery, len(tq.wheres), bindings) + ")" + ")",
	})
	return tq
}

func (tq *TableQuery) WhereGroup(fn func(*TableQuery)) *TableQuery {
	group := &TableQuery{}
	fn(group)
	groupSQL := buildWhereClause(group.wheres, false)
	groupBindings := flattenBindings(group.wheres)
	tq.wheres = append(tq.wheres, WhereCondition{
		Field: "(" + strings.ReplaceAll(groupSQL, "$w", "?") + ")",
		Value: groupBindings,
	})
	return tq
}

func (tq *TableQuery) OrderBy(field string) *TableQuery {
	tq.orders = append(tq.orders, OrderBy{Field: field, Desc: false})
	return tq
}

func (tq *TableQuery) OrderByDesc(field string) *TableQuery {
	tq.orders = append(tq.orders, OrderBy{Field: field, Desc: true})
	return tq
}

func (tq *TableQuery) OrderByRaw(sql string) *TableQuery {
	tq.orderRaws = append(tq.orderRaws, sql)
	return tq
}

func (tq *TableQuery) Limit(n int) *TableQuery {
	tq.limitVal = n
	return tq
}

func (tq *TableQuery) Offset(n int) *TableQuery {
	tq.offsetVal = n
	return tq
}

func (tq *TableQuery) AllowAll() *TableQuery {
	tq.allowAll = true
	return tq
}

func (tq *TableQuery) Join(table string, left string, op string, right string) *TableQuery {
	tq.joins = append(tq.joins, JoinCondition{
		Type:    "INNER",
		Table:   table,
		OnLeft:  left,
		OnRight: right,
		OnOp:    op,
	})
	return tq
}

func (tq *TableQuery) LeftJoin(table string, left string, op string, right string) *TableQuery {
	tq.joins = append(tq.joins, JoinCondition{
		Type:    "LEFT",
		Table:   table,
		OnLeft:  left,
		OnRight: right,
		OnOp:    op,
	})
	return tq
}

func (tq *TableQuery) RightJoin(table string, left string, op string, right string) *TableQuery {
	tq.joins = append(tq.joins, JoinCondition{
		Type:    "RIGHT",
		Table:   table,
		OnLeft:  left,
		OnRight: right,
		OnOp:    op,
	})
	return tq
}

func (tq *TableQuery) buildSQL() string {
	var sql strings.Builder
	sql.WriteString("SELECT ")
	selectParts := append([]string{}, tq.selects...)
	selectParts = append(selectParts, tq.selectRaws...)
	if len(selectParts) > 0 {
		sql.WriteString(formatSelects(selectParts))
	} else {
		sql.WriteString("*")
	}
	sql.WriteString(" FROM " + tq.table)

	for _, join := range tq.joins {
		fmt.Fprintf(&sql, " %s JOIN %s ON %s %s %s", join.Type, join.Table, join.OnLeft, join.OnOp, join.OnRight)
	}

	if len(tq.wheres) > 0 {
		sql.WriteString(" WHERE ")
		sql.WriteString(buildWhereClause(tq.wheres, false))
	}

	if len(tq.orders) > 0 || len(tq.orderRaws) > 0 {
		orderParts := make([]string, 0, len(tq.orders)+len(tq.orderRaws))
		for _, o := range tq.orders {
			part := o.Field
			if o.Desc {
				part += " DESC"
			}
			orderParts = append(orderParts, part)
		}
		orderParts = append(orderParts, tq.orderRaws...)
		sql.WriteString(" ORDER BY " + strings.Join(orderParts, ", "))
	}

	if tq.limitVal > 0 {
		fmt.Fprintf(&sql, " LIMIT %d", tq.limitVal)
	}

	if tq.offsetVal > 0 {
		fmt.Fprintf(&sql, " OFFSET %d", tq.offsetVal)
	}

	return sql.String()
}

func (tq *TableQuery) String() string {
	return tq.buildSQL()
}

func (tq *TableQuery) SQL() (string, map[string]any) {
	return tq.buildSQL(), tq.buildParams()
}

func (tq *TableQuery) All(ctx context.Context, results any) error {
	if err := ensureTableName(tq.table); err != nil {
		return err
	}
	sql, params := tq.buildSQL(), tq.buildParams()
	resp, err := tq.db.query(ctx, sql, params)
	if err != nil {
		return err
	}

	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil {
		return nil
	}

	return mapSliceToStruct((*resp)[0].Result, results)
}

func (tq *TableQuery) One(ctx context.Context, result any) error {
	if err := ensureTableName(tq.table); err != nil {
		return err
	}
	tq.limitVal = 1
	sql, params := tq.buildSQL(), tq.buildParams()
	resp, err := tq.db.query(ctx, sql, params)
	if err != nil {
		return err
	}

	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil || len((*resp)[0].Result) == 0 {
		return ErrNotFound
	}

	return mapToStruct((*resp)[0].Result[0], result)
}

func (tq *TableQuery) Count(ctx context.Context) (int, error) {
	if err := ensureTableName(tq.table); err != nil {
		return 0, err
	}
	sql, params := tq.buildSQL(), tq.buildParams()
	countSQL := "SELECT count() as count FROM (" + sql + ") as subquery"

	resp, err := tq.db.query(ctx, countSQL, params)
	if err != nil {
		return 0, err
	}

	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil || len((*resp)[0].Result) == 0 {
		return 0, nil
	}

	if count, ok := (*resp)[0].Result[0]["count"].(float64); ok {
		return int(count), nil
	}
	return 0, nil
}

func (tq *TableQuery) buildParams() map[string]any {
	return buildWhereParams(tq.wheres)
}

func (tq *TableQuery) First(ctx context.Context, result any) error {
	return tq.Limit(1).One(ctx, result)
}

func (tq *TableQuery) FirstOrFail(ctx context.Context, result any) error {
	return tq.First(ctx, result)
}

func (tq *TableQuery) Exists(ctx context.Context) (bool, error) {
	count, err := tq.Count(ctx)
	return count > 0, err
}

func (tq *TableQuery) Insert(ctx context.Context, data any) error {
	if err := ensureTableName(tq.table); err != nil {
		return err
	}
	content, err := structToMap(data)
	if err != nil {
		return err
	}
	_, err = createRecord[any](ctx, tq.db, models.Table(tq.table), content)
	return err
}

func (tq *TableQuery) Update(ctx context.Context, data any) error {
	if err := ensureTableName(tq.table); err != nil {
		return err
	}
	if err := ensureSafeMutation(tq.allowAll, tq.wheres); err != nil {
		return err
	}
	content, err := updateContentMap(data)
	if err != nil {
		return err
	}
	setSQL, setParams := buildUpdateSetClause(content)
	params := mergeParams(tq.buildParams(), setParams)
	sql := "UPDATE " + tq.table + " SET " + setSQL + " WHERE " + extractWhereClause(tq.wheres, false)
	_, err = tq.db.execQuery(ctx, sql, params)
	return err
}

func (tq *TableQuery) Delete(ctx context.Context) error {
	if err := ensureTableName(tq.table); err != nil {
		return err
	}
	if err := ensureSafeMutation(tq.allowAll, tq.wheres); err != nil {
		return err
	}
	params := tq.buildParams()
	sql := "DELETE FROM " + tq.table + " WHERE " + extractWhereClause(tq.wheres, false)
	_, err := tq.db.execQuery(ctx, sql, params)
	return err
}

func (tq *TableQuery) InsertMany(ctx context.Context, rows []map[string]any) error {
	for _, row := range rows {
		if err := tq.Insert(ctx, row); err != nil {
			return err
		}
	}
	return nil
}

func (tq *TableQuery) UpdateMany(ctx context.Context, rows []map[string]any) error {
	for _, row := range rows {
		if err := tq.Update(ctx, row); err != nil {
			return err
		}
	}
	return nil
}

func extractWhereClause(wheres []WhereCondition, stringifyID bool) string {
	if len(wheres) == 0 {
		return "1=1"
	}
	return buildWhereClause(wheres, stringifyID)
}

func mapToStruct(m map[string]any, s any) error {
	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return fmt.Errorf("expected struct, got %s", v.Kind())
	}

	meta := getModelMetadata(v.Type())
	for _, field := range meta.columns {
		val, ok := m[field.Column]
		if !ok {
			continue
		}

		fieldVal := valueByIndexPath(v, field.IndexPath)
		if !fieldVal.CanSet() {
			continue
		}

		if field.Column == "id" {
			if recordID, ok := val.(models.RecordID); ok {
				fieldVal.Set(reflect.ValueOf(recordID))
				continue
			}
			if str, ok := val.(string); ok {
				parts := strings.Split(str, ":")
				if len(parts) == 2 {
					fieldVal.Set(reflect.ValueOf(models.NewRecordID(parts[0], parts[1])))
				}
				continue
			}
			if mMap, ok := val.(map[string]any); ok {
				if table, ok := mMap["Table"].(string); ok {
					if idVal, ok := mMap["ID"]; ok {
						fieldVal.Set(reflect.ValueOf(models.NewRecordID(table, fmt.Sprintf("%v", idVal))))
					}
				}
				continue
			}
		}

		setValue(fieldVal, val)
	}

	return nil
}

func setValue(fieldVal reflect.Value, val any) {
	if val == nil {
		return
	}

	fieldType := fieldVal.Type()
	valType := reflect.TypeOf(val)

	if fieldType == valType {
		fieldVal.Set(reflect.ValueOf(val))
		return
	}

	switch fieldType.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fieldVal.SetInt(int64(toNumeric(val)))
	case reflect.Float32, reflect.Float64:
		fieldVal.SetFloat(toFloat(val))
	case reflect.Bool:
		fieldVal.SetBool(toBool(val))
	case reflect.String:
		fieldVal.SetString(toString(val))
	default:
		fieldVal.Set(reflect.ValueOf(val))
	}
}

func toNumeric(val any) int64 {
	switch v := val.(type) {
	case int:
		return int64(v)
	case int8:
		return int64(v)
	case int16:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	case float32:
		return int64(v)
	case uint:
		return int64(v)
	case uint8:
		return int64(v)
	case uint16:
		return int64(v)
	case uint32:
		return int64(v)
	case uint64:
		return int64(v)
	default:
		return 0
	}
}

func toFloat(val any) float64 {
	switch v := val.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int, int8, int16, int32, int64:
		return float64(reflect.ValueOf(v).Int())
	case uint, uint8, uint16, uint32, uint64:
		return float64(reflect.ValueOf(v).Uint())
	default:
		return 0
	}
}

func toBool(val any) bool {
	switch v := val.(type) {
	case bool:
		return v
	default:
		return false
	}
}

func toString(val any) string {
	switch v := val.(type) {
	case string:
		return v
	default:
		return fmt.Sprintf("%v", val)
	}
}

func structToMap(s any) (map[string]any, error) {
	result := make(map[string]any)

	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %s", v.Kind())
	}

	meta := getModelMetadata(v.Type())
	for _, field := range meta.columns {
		if field.Auto {
			continue
		}

		fieldValue := valueByIndexPath(v, field.IndexPath)
		if fieldValue.Kind() == reflect.Struct {
			if fieldValue.Type() == reflect.TypeOf(time.Time{}) {
				ts := fieldValue.Interface().(time.Time)
				if ts.IsZero() {
					continue
				}
			}
		}

		value := fieldValue.Interface()
		result[field.Column] = value
	}

	return result, nil
}

func updateContentMap(data any) (map[string]any, error) {
	switch v := data.(type) {
	case map[string]any:
		filtered := make(map[string]any, len(v))
		for key, value := range v {
			if value != nil {
				filtered[key] = value
			}
		}
		return filtered, nil
	default:
		content, err := structToMap(data)
		if err != nil {
			return nil, err
		}

		filtered := make(map[string]any, len(content))
		for key, value := range content {
			if value != nil && key != "id" && key != "created_at" {
				filtered[key] = value
			}
		}
		return filtered, nil
	}
}

func formatSelects(fields []string) string {
	var result strings.Builder
	for i, f := range fields {
		if i > 0 {
			result.WriteString(", ")
		}
		result.WriteString(f)
	}
	return result.String()
}

func GetTableName[T any](model *T) string {
	if tn, ok := any(model).(TableName); ok && tn.TableName() != "" {
		return tn.TableName()
	}
	name := reflect.TypeFor[T]().Name()
	if name == "" {
		return ""
	}
	return pluralize(toSnakeCase(name))
}

func QueryModel[T any](ctx context.Context, db *DB) *ModelQuery[T] {
	return &ModelQuery[T]{
		ctx:        ctx,
		db:         db,
		table:      GetTableName(new(T)),
		wheres:     []WhereCondition{},
		orders:     []OrderBy{},
		softDelete: modelSupportsSoftDelete[T](),
	}
}

type ModelQuery[T any] struct {
	ctx         context.Context
	db          *DB
	table       string
	selects     []string
	selectRaws  []string
	wheres      []WhereCondition
	orders      []OrderBy
	orderRaws   []string
	limitVal    int
	offsetVal   int
	withs       []string
	softDelete  bool
	withTrashed bool
	onlyTrashed bool
}

func (mq *ModelQuery[T]) With(relations ...string) *ModelQuery[T] {
	mq.withs = append(mq.withs, relations...)
	return mq
}

func (mq *ModelQuery[T]) WithTrashed() *ModelQuery[T] {
	mq.withTrashed = true
	return mq
}

func (mq *ModelQuery[T]) OnlyTrashed() *ModelQuery[T] {
	mq.onlyTrashed = true
	return mq
}

func (mq *ModelQuery[T]) OrWhere(field string, op string, value any) *ModelQuery[T] {
	mq.wheres = append(mq.wheres, WhereCondition{
		Field:    field,
		Operator: op,
		Value:    value,
		Or:       true,
	})
	return mq
}

func (mq *ModelQuery[T]) WhereNull(field string) *ModelQuery[T] {
	mq.wheres = append(mq.wheres, WhereCondition{
		Field:    field,
		Operator: "IS",
		Value:    nil,
	})
	return mq
}

func (mq *ModelQuery[T]) WhereNotNull(field string) *ModelQuery[T] {
	mq.wheres = append(mq.wheres, WhereCondition{
		Field:    field,
		Operator: "IS NOT",
		Value:    nil,
	})
	return mq
}

func (mq *ModelQuery[T]) Select(fields ...string) *ModelQuery[T] {
	mq.selects = fields
	return mq
}

func (mq *ModelQuery[T]) SelectRaw(sql string) *ModelQuery[T] {
	mq.selectRaws = append(mq.selectRaws, sql)
	return mq
}

func (mq *ModelQuery[T]) Where(field string, op string, value any) *ModelQuery[T] {
	mq.wheres = append(mq.wheres, WhereCondition{
		Field:    field,
		Operator: op,
		Value:    value,
	})
	return mq
}

func (mq *ModelQuery[T]) WhereRaw(sql string, bindings ...any) *ModelQuery[T] {
	mq.wheres = append(mq.wheres, WhereCondition{
		Field:    "(" + sql + ")",
		Operator: "",
		Value:    bindings,
	})
	return mq
}

func (mq *ModelQuery[T]) WhereIn(field string, values []any) *ModelQuery[T] {
	mq.wheres = append(mq.wheres, WhereCondition{
		Field:    field,
		Operator: "IN",
		Value:    values,
	})
	return mq
}

func (mq *ModelQuery[T]) WhereLike(field string, value string) *ModelQuery[T] {
	mq.wheres = append(mq.wheres, WhereCondition{
		Field:    field,
		Operator: "CONTAINS",
		Value:    value,
	})
	return mq
}

func (mq *ModelQuery[T]) WhereExists(subquery string, bindings ...any) *ModelQuery[T] {
	mq.wheres = append(mq.wheres, WhereCondition{
		Field: "(" + "EXISTS(" + formatRawCondition(subquery, len(mq.wheres), bindings) + ")" + ")",
	})
	return mq
}

func (mq *ModelQuery[T]) WhereGroup(fn func(*ModelQuery[T])) *ModelQuery[T] {
	group := &ModelQuery[T]{}
	fn(group)
	groupSQL := buildWhereClause(group.wheres, false)
	groupBindings := flattenBindings(group.wheres)
	mq.wheres = append(mq.wheres, WhereCondition{
		Field: "(" + strings.ReplaceAll(groupSQL, "$w", "?") + ")",
		Value: groupBindings,
	})
	return mq
}

func (mq *ModelQuery[T]) OrderBy(field string) *ModelQuery[T] {
	mq.orders = append(mq.orders, OrderBy{Field: field, Desc: false})
	return mq
}

func (mq *ModelQuery[T]) OrderByDesc(field string) *ModelQuery[T] {
	mq.orders = append(mq.orders, OrderBy{Field: field, Desc: true})
	return mq
}

func (mq *ModelQuery[T]) OrderByRaw(sql string) *ModelQuery[T] {
	mq.orderRaws = append(mq.orderRaws, sql)
	return mq
}

func (mq *ModelQuery[T]) Limit(n int) *ModelQuery[T] {
	mq.limitVal = n
	return mq
}

func (mq *ModelQuery[T]) Offset(n int) *ModelQuery[T] {
	mq.offsetVal = n
	return mq
}

func (mq *ModelQuery[T]) buildSQL() string {
	var sql strings.Builder
	sql.WriteString("SELECT ")
	selectParts := append([]string{}, mq.selects...)
	selectParts = append(selectParts, mq.selectRaws...)
	if len(selectParts) > 0 {
		sql.WriteString(formatSelects(selectParts))
	} else {
		sql.WriteString("*")
	}
	sql.WriteString(" FROM " + mq.table)

	if len(mq.wheres) > 0 {
		sql.WriteString(" WHERE ")
		sql.WriteString(buildWhereClause(mq.wheres, false))
	}

	appendSoftDeleteClause(&sql, len(mq.wheres) > 0, mq.softDelete, mq.withTrashed, mq.onlyTrashed)

	if len(mq.orders) > 0 || len(mq.orderRaws) > 0 {
		orderParts := make([]string, 0, len(mq.orders)+len(mq.orderRaws))
		for _, o := range mq.orders {
			part := o.Field
			if o.Desc {
				part += " DESC"
			}
			orderParts = append(orderParts, part)
		}
		orderParts = append(orderParts, mq.orderRaws...)
		sql.WriteString(" ORDER BY " + strings.Join(orderParts, ", "))
	}

	if mq.limitVal > 0 {
		fmt.Fprintf(&sql, " LIMIT %d", mq.limitVal)
	}

	if mq.offsetVal > 0 {
		fmt.Fprintf(&sql, " START %d", mq.offsetVal)
	}

	return sql.String()
}

func (mq *ModelQuery[T]) buildParams() map[string]any {
	return buildWhereParams(mq.wheres)
}

func (mq *ModelQuery[T]) All(ctx context.Context, results *[]T) error {
	if err := ensureTableName(mq.table); err != nil {
		return err
	}
	sql := mq.buildSQL()
	resp, err := mq.db.query(ctx, sql, mq.buildParams())
	if err != nil {
		return err
	}

	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil {
		return nil
	}

	if err := mapSliceToStruct((*resp)[0].Result, results); err != nil {
		return err
	}
	if len(mq.withs) > 0 {
		return mq.db.With(ctx, results, mq.withs...)
	}
	return nil
}

func (mq *ModelQuery[T]) One(ctx context.Context, result *T) error {
	if err := ensureTableName(mq.table); err != nil {
		return err
	}
	mq.limitVal = 1
	sql := mq.buildSQL()
	resp, err := mq.db.query(ctx, sql, mq.buildParams())
	if err != nil {
		return err
	}

	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil || len((*resp)[0].Result) == 0 {
		return ErrNotFound
	}

	if err := mapToStruct((*resp)[0].Result[0], result); err != nil {
		return err
	}
	if len(mq.withs) > 0 {
		return mq.db.With(ctx, result, mq.withs...)
	}
	return nil
}

func (mq *ModelQuery[T]) First(ctx context.Context, result *T) error {
	return mq.Limit(1).One(ctx, result)
}

func (mq *ModelQuery[T]) Count(ctx context.Context) (int, error) {
	if err := ensureTableName(mq.table); err != nil {
		return 0, err
	}
	sql := "SELECT count() as count FROM (" + mq.buildSQL() + ") as subquery"
	resp, err := mq.db.query(ctx, sql, mq.buildParams())
	if err != nil {
		return 0, err
	}

	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil || len((*resp)[0].Result) == 0 {
		return 0, nil
	}

	if count, ok := (*resp)[0].Result[0]["count"].(float64); ok {
		return int(count), nil
	}
	return 0, nil
}

func (mq *ModelQuery[T]) Find(ctx context.Context, id string, result *T) error {
	if err := ensureTableName(mq.table); err != nil {
		return err
	}
	sql := "SELECT * FROM " + mq.table + " WHERE id = $id LIMIT 1"
	resp, err := mq.db.query(ctx, sql, map[string]any{"id": id})
	if err != nil {
		return err
	}

	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil || len((*resp)[0].Result) == 0 {
		return ErrNotFound
	}

	if err := mapToStruct((*resp)[0].Result[0], result); err != nil {
		return err
	}
	if len(mq.withs) > 0 {
		return mq.db.With(ctx, result, mq.withs...)
	}
	return nil
}

func (mq *ModelQuery[T]) FindOne(ctx context.Context, result *T) error {
	if err := ensureTableName(mq.table); err != nil {
		return err
	}
	mq.limitVal = 1
	sql := mq.buildSQL()
	resp, err := mq.db.query(ctx, sql, mq.buildParams())
	if err != nil {
		return err
	}

	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil || len((*resp)[0].Result) == 0 {
		return ErrNotFound
	}

	if err := mapToStruct((*resp)[0].Result[0], result); err != nil {
		return err
	}
	if len(mq.withs) > 0 {
		return mq.db.With(ctx, result, mq.withs...)
	}
	return nil
}

func (mq *ModelQuery[T]) FirstOrFail(ctx context.Context, result *T) error {
	return mq.First(ctx, result)
}

func (mq *ModelQuery[T]) FindOrFail(ctx context.Context, id string, result *T) error {
	return mq.Find(ctx, id, result)
}

func Create[T any](ctx context.Context, db *DB, model *T) error {
	if err := callBeforeCreate(ctx, db, model); err != nil {
		return err
	}
	content, err := structToMap(model)
	if err != nil {
		return err
	}

	filtered := make(map[string]any)
	for k, v := range content {
		if v != nil {
			filtered[k] = v
		}
	}

	table := GetTableName(model)

	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	idField := v.FieldByName("ID")
	if idField.IsValid() && !idField.IsZero() {
		idVal := idField.Interface()
		if rid, ok := idVal.(models.RecordID); ok && rid.String() != "" {
			existing, _ := selectRecord[any](ctx, db, rid)
			if existing != nil {
				_, err = updateRecord[any](ctx, db, rid, filtered)
				if err != nil {
					return err
				}
				return callAfterUpdate(ctx, db, model)
			}
			_, err = createRecord[any](ctx, db, rid, filtered)
			if err != nil {
				return err
			}
			return callAfterCreate(ctx, db, model)
		}
	}

	now := time.Now()
	filtered["created_at"] = now
	filtered["updated_at"] = now

	_, err = createRecord[any](ctx, db, models.Table(table), filtered)
	if err != nil {
		return err
	}
	return callAfterCreate(ctx, db, model)
}

func Update[T any](ctx context.Context, db *DB, model *T) error {
	if err := callBeforeUpdate(ctx, db, model); err != nil {
		return err
	}
	content, err := structToMap(model)
	if err != nil {
		return err
	}

	filtered := make(map[string]any)
	for k, v := range content {
		if v != nil && k != "id" && k != "created_at" {
			filtered[k] = v
		}
	}

	filtered["updated_at"] = time.Now()

	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	idField := v.FieldByName("ID")
	if idField.IsValid() && !idField.IsZero() {
		idVal := idField.Interface()
		if rid, ok := idVal.(models.RecordID); ok && rid.String() != "" {
			_, err = updateRecord[any](ctx, db, rid, filtered)
			if err != nil {
				return err
			}
			return callAfterUpdate(ctx, db, model)
		}
	}

	table := GetTableName(model)
	setSQL, params := buildUpdateSetClause(filtered)
	sql := "UPDATE " + table + " SET " + setSQL
	_, err = db.execQuery(ctx, sql, params)
	if err != nil {
		return err
	}
	return callAfterUpdate(ctx, db, model)
}

func Delete[T any](ctx context.Context, db *DB, model *T) error {
	if err := callBeforeDelete(ctx, db, model); err != nil {
		return err
	}
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	idField := v.FieldByName("ID")
	if !idField.IsValid() {
		return ErrInvalidModel
	}

	idVal := idField.Interface()
	rid, ok := idVal.(models.RecordID)
	if !ok {
		return ErrInvalidModel
	}

	_, err := deleteRecord[any](ctx, db, rid)
	if err != nil {
		return err
	}
	return callAfterDelete(ctx, db, model)
}

func All[T any](ctx context.Context, db *DB, results *[]T) error {
	table := GetTableName(new(T))
	if err := ensureTableName(table); err != nil {
		return err
	}
	resp, err := db.query(ctx, "SELECT * FROM "+table, nil)
	if err != nil {
		return err
	}
	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil || len((*resp)[0].Result) == 0 {
		return nil
	}
	return mapSliceToStruct((*resp)[0].Result, results)
}

func Find[T any](ctx context.Context, db *DB, id string, result *T) error {
	table := GetTableName(new(T))
	if err := ensureTableName(table); err != nil {
		return err
	}
	resp, err := db.query(ctx, "SELECT * FROM "+table+" WHERE string::concat(id) = $id LIMIT 1", map[string]any{"id": id})
	if err != nil {
		return err
	}
	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil || len((*resp)[0].Result) == 0 {
		return ErrNotFound
	}
	return mapToStruct((*resp)[0].Result[0], result)
}
