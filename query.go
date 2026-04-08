package surrealgoorm

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/surrealdb/surrealdb.go"
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
	_, err := surrealdb.Query[any](ctx, rb.db.raw, rb.sql, rb.buildParams())
	return err
}

func (rb *RawBuilder) All(ctx context.Context, results any) error {
	resp, err := surrealdb.Query[[]map[string]any](ctx, rb.db.raw, rb.sql, rb.buildParams())
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

	resp, err := surrealdb.Query[[]map[string]any](ctx, rb.db.raw, rb.sql, rb.buildParams())
	if err != nil {
		return err
	}

	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil || len((*resp)[0].Result) == 0 {
		return fmt.Errorf("record not found")
	}

	return mapToStruct((*resp)[0].Result[0], result)
}

func (rb *RawBuilder) Scalar(ctx context.Context) (any, error) {
	resp, err := surrealdb.Query[[]map[string]any](ctx, rb.db.raw, rb.sql, rb.buildParams())
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
	if len(sql) > 10 && sql[len(sql)-10:] != "LIMIT 1" {
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
	db        *DB
	table     string
	selects   []string
	wheres    []WhereCondition
	orders    []OrderBy
	limitVal  int
	offsetVal int
	joins     []JoinCondition
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

func (tq *TableQuery) OrderBy(field string) *TableQuery {
	tq.orders = append(tq.orders, OrderBy{Field: field, Desc: false})
	return tq
}

func (tq *TableQuery) OrderByDesc(field string) *TableQuery {
	tq.orders = append(tq.orders, OrderBy{Field: field, Desc: true})
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
	if len(tq.selects) > 0 {
		sql.WriteString(formatSelects(tq.selects))
	} else {
		sql.WriteString("*")
	}
	sql.WriteString(" FROM " + tq.table)

	for _, join := range tq.joins {
		fmt.Fprintf(&sql, " %s JOIN %s ON %s %s %s", join.Type, join.Table, join.OnLeft, join.OnOp, join.OnRight)
	}

	if len(tq.wheres) > 0 {
		sql.WriteString(" WHERE ")
		for i, w := range tq.wheres {
			if i > 0 {
				sql.WriteString(" AND ")
			}
			if w.Operator != "" {
				fmt.Fprintf(&sql, "%s %s $w%d", w.Field, w.Operator, i)
			} else {
				sql.WriteString(w.Field)
			}
		}
	}

	for _, o := range tq.orders {
		sql.WriteString(" ORDER BY " + o.Field)
		if o.Desc {
			sql.WriteString(" DESC")
		}
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
	sql, params := tq.buildSQL(), tq.buildParams()
	resp, err := surrealdb.Query[[]map[string]any](ctx, tq.db.raw, sql, params)
	if err != nil {
		return err
	}

	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil {
		return nil
	}

	return mapSliceToStruct((*resp)[0].Result, results)
}

func (tq *TableQuery) One(ctx context.Context, result any) error {
	tq.limitVal = 1
	sql, params := tq.buildSQL(), tq.buildParams()
	resp, err := surrealdb.Query[[]map[string]any](ctx, tq.db.raw, sql, params)
	if err != nil {
		return err
	}

	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil || len((*resp)[0].Result) == 0 {
		return fmt.Errorf("record not found")
	}

	return mapToStruct((*resp)[0].Result[0], result)
}

func (tq *TableQuery) Count(ctx context.Context) (int, error) {
	sql, params := tq.buildSQL(), tq.buildParams()
	countSQL := "SELECT count() as count FROM (" + sql + ") as subquery"

	resp, err := surrealdb.Query[[]map[string]any](ctx, tq.db.raw, countSQL, params)
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
	params := make(map[string]any)

	for i, w := range tq.wheres {
		if w.Operator != "" {
			params[fmt.Sprintf("w%d", i)] = w.Value
		}
	}

	return params
}

func (tq *TableQuery) First(ctx context.Context, result any) error {
	return tq.Limit(1).One(ctx, result)
}

func (tq *TableQuery) Exists(ctx context.Context) (bool, error) {
	count, err := tq.Count(ctx)
	return count > 0, err
}

func (tq *TableQuery) Insert(ctx context.Context, data any) error {
	content, err := structToMap(data)
	if err != nil {
		return err
	}
	_, err = surrealdb.Create[any](ctx, tq.db.raw, models.Table(tq.table), content)
	return err
}

func (tq *TableQuery) Update(ctx context.Context, data any) error {
	content, err := structToMap(data)
	if err != nil {
		return err
	}
	_ /* sql*/, params := tq.buildSQL(), tq.buildParams()
	sql := "UPDATE " + tq.table + " SET " + formatUpdateSet(content) + " WHERE " + extractWhereClause(tq.wheres)
	_, err = surrealdb.Query[any](ctx, tq.db.raw, sql, params)
	return err
}

func (tq *TableQuery) Delete(ctx context.Context) error {
	_ /* sql*/, params := tq.buildSQL(), tq.buildParams()
	sql := "DELETE FROM " + tq.table + " WHERE " + extractWhereClause(tq.wheres)
	_, err := surrealdb.Query[any](ctx, tq.db.raw, sql, params)
	return err
}

func formatUpdateSet(data map[string]any) string {
	var result strings.Builder
	i := 0
	for k := range data {
		if i > 0 {
			result.WriteString(", ")
		}
		fmt.Fprintf(&result, "%s = $u%d", k, i)
		i++
	}
	return result.String()
}

func extractWhereClause(wheres []WhereCondition) string {
	if len(wheres) == 0 {
		return "1=1"
	}
	var sql strings.Builder
	for i, w := range wheres {
		if i > 0 {
			sql.WriteString(" AND ")
		}
		fmt.Fprintf(&sql, "%s %s $w%d", w.Field, w.Operator, i)
	}
	return sql.String()
}

func mapToStruct(m map[string]any, s any) error {
	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return fmt.Errorf("expected struct, got %s", v.Kind())
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("orm")
		if tag == "" {
			continue
		}

		parts := splitTag(tag)
		columnName := parts[0]

		if columnName == "" || columnName == "-" {
			continue
		}

		if val, ok := m[columnName]; ok {
			fieldVal := v.Field(i)
			if fieldVal.CanSet() {
				fieldVal.Set(reflect.ValueOf(val))
			}
		}
	}

	return nil
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

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("orm")
		if tag == "" {
			continue
		}

		parts := splitTag(tag)
		columnName := parts[0]

		if columnName == "" || columnName == "-" {
			continue
		}

		value := v.Field(i).Interface()
		result[columnName] = value
	}

	return result, nil
}

func splitTag(tag string) []string {
	var parts []string
	current := ""
	for _, c := range tag {
		if c == ';' || c == ':' {
			if current != "" {
				parts = append(parts, current)
			}
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
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
	name := strings.TrimPrefix(reflect.TypeFor[T]().Name(), "")
	if name == "" {
		return ""
	}
	runes := []rune(name)
	runes[0] = rune(runes[0] + 32)
	return string(runes) + "s"
}

func QueryModel[T any](ctx context.Context, db *DB) *ModelQuery[T] {
	return &ModelQuery[T]{
		ctx:    ctx,
		db:     db,
		table:  GetTableName(new(T)),
		wheres: []WhereCondition{},
		orders: []OrderBy{},
	}
}

type ModelQuery[T any] struct {
	ctx         context.Context
	db          *DB
	table       string
	selects     []string
	wheres      []WhereCondition
	orders      []OrderBy
	limitVal    int
	offsetVal   int
	withs       []string
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

func (mq *ModelQuery[T]) OrderBy(field string) *ModelQuery[T] {
	mq.orders = append(mq.orders, OrderBy{Field: field, Desc: false})
	return mq
}

func (mq *ModelQuery[T]) OrderByDesc(field string) *ModelQuery[T] {
	mq.orders = append(mq.orders, OrderBy{Field: field, Desc: true})
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
	if len(mq.selects) > 0 {
		sql.WriteString(formatSelects(mq.selects))
	} else {
		sql.WriteString("*")
	}
	sql.WriteString(" FROM " + mq.table)

	if len(mq.wheres) > 0 {
		sql.WriteString(" WHERE ")
		for i, w := range mq.wheres {
			if i > 0 && !w.Or {
				sql.WriteString(" AND ")
			}
			if i > 0 && w.Or {
				sql.WriteString(" OR ")
			}
			if w.Operator != "" {
				fmt.Fprintf(&sql, "%s %s $w%d", w.Field, w.Operator, i)
			} else {
				sql.WriteString(w.Field)
			}
		}
	}

	if mq.onlyTrashed {
		if len(mq.wheres) > 0 {
			sql.WriteString(" AND deleted_at IS NOT NULL")
		} else {
			sql.WriteString(" WHERE deleted_at IS NOT NULL")
		}
	} else if !mq.withTrashed {
		if len(mq.wheres) > 0 {
			sql.WriteString(" AND deleted_at IS NULL")
		} else {
			sql.WriteString(" WHERE deleted_at IS NULL")
		}
	}

	for _, o := range mq.orders {
		sql.WriteString(" ORDER BY " + o.Field)
		if o.Desc {
			sql.WriteString(" DESC")
		}
	}

	if mq.limitVal > 0 {
		fmt.Fprintf(&sql, " LIMIT %d", mq.limitVal)
	}

	if mq.offsetVal > 0 {
		fmt.Fprintf(&sql, " OFFSET %d", mq.offsetVal)
	}

	return sql.String()
}

func (mq *ModelQuery[T]) buildParams() map[string]any {
	params := make(map[string]any)

	for i, w := range mq.wheres {
		if w.Operator != "" {
			params[fmt.Sprintf("w%d", i)] = w.Value
		}
	}

	return params
}

func (mq *ModelQuery[T]) All(ctx context.Context, results *[]T) error {
	sql := mq.buildSQL()
	if len(mq.withs) > 0 {
		sql = strings.Replace(sql, "SELECT *", "SELECT * FETCH "+strings.Join(mq.withs, ", "), 1)
	}
	resp, err := surrealdb.Query[[]map[string]any](ctx, mq.db.raw, sql, mq.buildParams())
	if err != nil {
		return err
	}

	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil {
		return nil
	}

	return mapSliceToStruct((*resp)[0].Result, results)
}

func (mq *ModelQuery[T]) One(ctx context.Context, result *T) error {
	mq.limitVal = 1
	sql := mq.buildSQL()
	if len(mq.withs) > 0 {
		sql = strings.Replace(sql, "SELECT *", "SELECT * FETCH "+strings.Join(mq.withs, ", "), 1)
	}
	resp, err := surrealdb.Query[[]map[string]any](ctx, mq.db.raw, sql, mq.buildParams())
	if err != nil {
		return err
	}

	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil || len((*resp)[0].Result) == 0 {
		return fmt.Errorf("record not found")
	}

	return mapToStruct((*resp)[0].Result[0], result)
}

func (mq *ModelQuery[T]) First(ctx context.Context, result *T) error {
	return mq.Limit(1).One(ctx, result)
}

func (mq *ModelQuery[T]) Count(ctx context.Context) (int, error) {
	sql := "SELECT count() as count FROM (" + mq.buildSQL() + ") as subquery"
	resp, err := surrealdb.Query[[]map[string]any](ctx, mq.db.raw, sql, mq.buildParams())
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
	sql := "SELECT * FROM " + mq.table + " WHERE id = $id LIMIT 1"
	if len(mq.withs) > 0 {
		sql = strings.Replace(sql, "SELECT *", "SELECT * FETCH "+strings.Join(mq.withs, ", "), 1)
	}
	resp, err := surrealdb.Query[[]map[string]any](ctx, mq.db.raw, sql, map[string]any{"id": id})
	if err != nil {
		return err
	}

	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil || len((*resp)[0].Result) == 0 {
		return fmt.Errorf("record not found")
	}

	return mapToStruct((*resp)[0].Result[0], result)
}

func Create[T any](ctx context.Context, db *DB, model *T) error {
	content, err := structToMap(model)
	if err != nil {
		return err
	}
	table := GetTableName(model)
	_, err = surrealdb.Create[any](ctx, db.raw, models.Table(table), content)
	return err
}

func Update[T any](ctx context.Context, db *DB, model *T) error {
	return nil
}

func Delete[T any](ctx context.Context, db *DB, model *T) error {
	return nil
}

func All[T any](ctx context.Context, db *DB, results *[]T) error {
	table := GetTableName(new(T))
	resp, err := surrealdb.Query[[]map[string]any](ctx, db.raw, "SELECT * FROM "+table, nil)
	if err != nil {
		return err
	}
	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil {
		return nil
	}
	return mapSliceToStruct((*resp)[0].Result, results)
}

func Find[T any](ctx context.Context, db *DB, id string, result *T) error {
	table := GetTableName(new(T))
	resp, err := surrealdb.Query[[]map[string]any](ctx, db.raw, "SELECT * FROM "+table+" WHERE id = $id LIMIT 1", map[string]any{"id": id})
	if err != nil {
		return err
	}
	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil || len((*resp)[0].Result) == 0 {
		return fmt.Errorf("record not found")
	}
	return mapToStruct((*resp)[0].Result[0], result)
}
