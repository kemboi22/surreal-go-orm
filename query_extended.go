package surrealgoorm

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"strings"

	"github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

type ScopeFunc func(*QueryBuilder) *QueryBuilder

type ScopeFuncModel[T any] func(*ModelQuery[T]) *ModelQuery[T]

func Scope[T any](fn ScopeFuncModel[T]) ScopeFuncModel[T] {
	return fn
}

func ApplyScopes[T any](query *ModelQuery[T], scopes []ScopeFuncModel[T]) *ModelQuery[T] {
	for _, scope := range scopes {
		query = scope(query)
	}
	return query
}

type QueryBuilder struct {
	db           *DB
	table        string
	selectFields []string
	wheres       []WhereCondition
	orders       []OrderBy
	limitVal     int
	offsetVal    int
	withs        []string
	groupBys     []string
	onlyTrashed  bool
	withTrashed  bool
}

func (db *DB) Query(table string) *QueryBuilder {
	return &QueryBuilder{
		db:     db,
		table:  table,
		wheres: []WhereCondition{},
		orders: []OrderBy{},
	}
}

func (qb *QueryBuilder) Select(fields ...string) *QueryBuilder {
	qb.selectFields = fields
	return qb
}

func (qb *QueryBuilder) Where(field string, op string, value any) *QueryBuilder {
	qb.wheres = append(qb.wheres, WhereCondition{Field: field, Operator: op, Value: value})
	return qb
}

func (qb *QueryBuilder) OrWhere(field string, op string, value any) *QueryBuilder {
	qb.wheres = append(qb.wheres, WhereCondition{Field: field, Operator: op, Value: value, Or: true})
	return qb
}

func (qb *QueryBuilder) WhereIn(field string, values []any) *QueryBuilder {
	qb.wheres = append(qb.wheres, WhereCondition{Field: field, Operator: "IN", Value: values})
	return qb
}

func (qb *QueryBuilder) WhereNull(field string) *QueryBuilder {
	qb.wheres = append(qb.wheres, WhereCondition{Field: field, Operator: "IS", Value: nil})
	return qb
}

func (qb *QueryBuilder) WhereNotNull(field string) *QueryBuilder {
	qb.wheres = append(qb.wheres, WhereCondition{Field: field, Operator: "IS NOT", Value: nil})
	return qb
}

func (qb *QueryBuilder) WhereBetween(field string, start, end any) *QueryBuilder {
	qb.wheres = append(qb.wheres, WhereCondition{Field: field, Operator: "BETWEEN", Value: []any{start, end}})
	return qb
}

func (qb *QueryBuilder) OrderBy(field string) *QueryBuilder {
	qb.orders = append(qb.orders, OrderBy{Field: field, Desc: false})
	return qb
}

func (qb *QueryBuilder) OrderByDesc(field string) *QueryBuilder {
	qb.orders = append(qb.orders, OrderBy{Field: field, Desc: true})
	return qb
}

func (qb *QueryBuilder) Limit(n int) *QueryBuilder {
	qb.limitVal = n
	return qb
}

func (qb *QueryBuilder) Offset(n int) *QueryBuilder {
	qb.offsetVal = n
	return qb
}

func (qb *QueryBuilder) With(relations ...string) *QueryBuilder {
	qb.withs = append(qb.withs, relations...)
	return qb
}

func (qb *QueryBuilder) GroupBy(fields ...string) *QueryBuilder {
	qb.groupBys = fields
	return qb
}

func (qb *QueryBuilder) WithTrashed() *QueryBuilder {
	qb.withTrashed = true
	return qb
}

func (qb *QueryBuilder) OnlyTrashed() *QueryBuilder {
	qb.onlyTrashed = true
	return qb
}

func (qb *QueryBuilder) buildSQL() string {
	var sql strings.Builder
	sql.WriteString("SELECT ")
	if len(qb.selectFields) > 0 {
		sql.WriteString(formatSelects(qb.selectFields))
	} else {
		sql.WriteString("*")
	}
	sql.WriteString(" FROM " + qb.table)

	if len(qb.withs) > 0 {
		sql.WriteString(" FETCH " + formatSelects(qb.withs))
	}

	if len(qb.wheres) > 0 {
		sql.WriteString(" WHERE ")
		for i, w := range qb.wheres {
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

	if qb.onlyTrashed {
		sql.WriteString(" AND deleted_at IS NOT NULL")
	} else if !qb.withTrashed {
		sql.WriteString(" AND deleted_at IS NULL")
	}

	if len(qb.groupBys) > 0 {
		sql.WriteString(" GROUP BY " + formatSelects(qb.groupBys))
	}

	for _, o := range qb.orders {
		sql.WriteString(" ORDER BY " + o.Field)
		if o.Desc {
			sql.WriteString(" DESC")
		}
	}

	if qb.limitVal > 0 {
		fmt.Fprintf(&sql, " LIMIT %d", qb.limitVal)
	}

	if qb.offsetVal > 0 {
		fmt.Fprintf(&sql, " START %d", qb.offsetVal)
	}

	return sql.String()
}

func (qb *QueryBuilder) buildParams() map[string]any {
	params := make(map[string]any)
	for i, w := range qb.wheres {
		if w.Operator == "IN" || w.Operator == "BETWEEN" {
			params[fmt.Sprintf("w%d", i)] = w.Value
		} else if w.Value != nil {
			params[fmt.Sprintf("w%d", i)] = w.Value
		}
	}
	return params
}

func (qb *QueryBuilder) SQL() (string, map[string]any) {
	return qb.buildSQL(), qb.buildParams()
}

func (qb *QueryBuilder) All(ctx context.Context, results any) error {
	sql, params := qb.buildSQL(), qb.buildParams()
	resp, err := surrealdb.Query[[]map[string]any](ctx, qb.db.raw, sql, params)
	if err != nil {
		return err
	}
	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil {
		return nil
	}
	return mapSliceToStruct((*resp)[0].Result, results)
}

func (qb *QueryBuilder) One(ctx context.Context, result any) error {
	qb.limitVal = 1
	sql, params := qb.buildSQL(), qb.buildParams()
	resp, err := surrealdb.Query[[]map[string]any](ctx, qb.db.raw, sql, params)
	if err != nil {
		return err
	}
	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil || len((*resp)[0].Result) == 0 {
		return fmt.Errorf("record not found")
	}
	return mapToStruct((*resp)[0].Result[0], result)
}

func (qb *QueryBuilder) First(ctx context.Context, result any) error {
	return qb.Limit(1).One(ctx, result)
}

func (qb *QueryBuilder) Count(ctx context.Context) (int, error) {
	sql, params := qb.buildSQL(), qb.buildParams()
	countSQL := "SELECT count() as count FROM (" + sql + ") as subquery"
	resp, err := surrealdb.Query[[]map[string]any](ctx, qb.db.raw, countSQL, params)
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

func (qb *QueryBuilder) Exists(ctx context.Context) (bool, error) {
	count, err := qb.Count(ctx)
	return count > 0, err
}

func (qb *QueryBuilder) Sum(ctx context.Context, field string) (float64, error) {
	sql := qb.buildSQL()
	params := qb.buildParams()
	sumSQL := fmt.Sprintf("SELECT math::sum(%s) as sum FROM (%s) as subquery", field, sql)
	resp, err := surrealdb.Query[[]map[string]any](ctx, qb.db.raw, sumSQL, params)
	if err != nil {
		return 0, err
	}
	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil || len((*resp)[0].Result) == 0 {
		return 0, nil
	}
	if sum, ok := (*resp)[0].Result[0]["sum"].(float64); ok {
		return sum, nil
	}
	return 0, nil
}

func (qb *QueryBuilder) Avg(ctx context.Context, field string) (float64, error) {
	sql := qb.buildSQL()
	params := qb.buildParams()
	avgSQL := fmt.Sprintf("SELECT math::mean(%s) as avg FROM (%s) as subquery", field, sql)
	resp, err := surrealdb.Query[[]map[string]any](ctx, qb.db.raw, avgSQL, params)
	if err != nil {
		return 0, err
	}
	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil || len((*resp)[0].Result) == 0 {
		return 0, nil
	}
	if avg, ok := (*resp)[0].Result[0]["avg"].(float64); ok {
		return avg, nil
	}
	return 0, nil
}

func (qb *QueryBuilder) Min(ctx context.Context, field string) (float64, error) {
	sql := qb.buildSQL()
	params := qb.buildParams()
	minSQL := fmt.Sprintf("SELECT math::min(%s) as min FROM (%s) as subquery", field, sql)
	resp, err := surrealdb.Query[[]map[string]any](ctx, qb.db.raw, minSQL, params)
	if err != nil {
		return 0, err
	}
	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil || len((*resp)[0].Result) == 0 {
		return 0, nil
	}
	if min, ok := (*resp)[0].Result[0]["min"].(float64); ok {
		return min, nil
	}
	return 0, nil
}

func (qb *QueryBuilder) Max(ctx context.Context, field string) (float64, error) {
	sql := qb.buildSQL()
	params := qb.buildParams()
	maxSQL := fmt.Sprintf("SELECT math::max(%s) as max FROM (%s) as subquery", field, sql)
	resp, err := surrealdb.Query[[]map[string]any](ctx, qb.db.raw, maxSQL, params)
	if err != nil {
		return 0, err
	}
	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil || len((*resp)[0].Result) == 0 {
		return 0, nil
	}
	if max, ok := (*resp)[0].Result[0]["max"].(float64); ok {
		return max, nil
	}
	return 0, nil
}

func (qb *QueryBuilder) Insert(ctx context.Context, data any) error {
	content, err := structToMap(data)
	if err != nil {
		return err
	}
	_, err = surrealdb.Create[any](ctx, qb.db.raw, qb.table, content)
	return err
}

func (qb *QueryBuilder) Update(ctx context.Context, data any) error {
	content, err := structToMap(data)
	if err != nil {
		return err
	}
	_ /* sql*/, params := qb.buildSQL(), qb.buildParams()
	sql := "UPDATE " + qb.table + " SET " + formatUpdateSet(content) + " WHERE " + extractWhereClause(qb.wheres)
	_, err = surrealdb.Query[any](ctx, qb.db.raw, sql, params)
	return err
}

func (qb *QueryBuilder) Delete(ctx context.Context) error {
	_ /* sql*/, params := qb.buildSQL(), qb.buildParams()
	sql := "DELETE FROM " + qb.table + " WHERE " + extractWhereClause(qb.wheres)
	_, err := surrealdb.Query[any](ctx, qb.db.raw, sql, params)
	return err
}

func (qb *QueryBuilder) ForceDelete(ctx context.Context) error {
	oldWithTrashed := qb.withTrashed
	oldOnlyTrashed := qb.onlyTrashed
	qb.withTrashed = true
	qb.onlyTrashed = true
	_ /* sql*/, params := qb.buildSQL(), qb.buildParams()
	sql := "DELETE FROM " + qb.table + " WHERE " + extractWhereClause(qb.wheres)
	_, err := surrealdb.Query[any](ctx, qb.db.raw, sql, params)
	qb.withTrashed = oldWithTrashed
	qb.onlyTrashed = oldOnlyTrashed
	return err
}

func (qb *QueryBuilder) Restore(ctx context.Context) error {
	_ /* sql*/, params := qb.buildSQL(), qb.buildParams()
	sql := "UPDATE " + qb.table + " SET deleted_at = null WHERE " + extractWhereClause(qb.wheres)
	_, err := surrealdb.Query[any](ctx, qb.db.raw, sql, params)
	return err
}

type PaginationResult[T any] struct {
	Data       []T `json:"data"`
	Total      int `json:"total"`
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	TotalPages int `json:"total_pages"`
}

func (qb *QueryBuilder) Paginate(ctx context.Context, page, perPage int, results any) (*PaginationResult[any], error) {
	qb.limitVal = perPage
	qb.offsetVal = (page - 1) * perPage

	total, err := qb.Count(ctx)
	if err != nil {
		return nil, err
	}

	sql, params := qb.buildSQL(), qb.buildParams()
	resp, err := surrealdb.Query[[]map[string]any](ctx, qb.db.raw, sql, params)
	if err != nil {
		return nil, err
	}

	var data []any
	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil {
		data = []any{}
	} else {
		data = make([]any, len((*resp)[0].Result))
		for i, row := range (*resp)[0].Result {
			data[i] = row
		}
	}

	totalPages := total / perPage
	if total%perPage != 0 {
		totalPages++
	}

	return &PaginationResult[any]{
		Data:       data,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	}, nil
}

func FirstOrCreate[T any](ctx context.Context, db *DB, model *T, where map[string]any) error {
	table := GetTableName(model)

	var conditions strings.Builder
	params := make(map[string]any)
	i := 0
	for field, value := range where {
		if i > 0 {
			conditions.WriteString(" AND ")
		}
		fmt.Fprintf(&conditions, "%s = $w%d", field, i)
		params[fmt.Sprintf("w%d", i)] = value
		i++
	}

	sql := "SELECT * FROM " + table + " WHERE " + conditions.String() + " LIMIT 1"
	resp, err := surrealdb.Query[[]map[string]any](ctx, db.raw, sql, params)
	if err != nil {
		return err
	}

	if resp != nil && len(*resp) > 0 && (*resp)[0].Result != nil && len((*resp)[0].Result) > 0 {
		return mapToStruct((*resp)[0].Result[0], model)
	}

	return Create(ctx, db, model)
}

func UpdateOrCreate[T any](ctx context.Context, db *DB, model *T, where map[string]any, update map[string]any) error {
	table := GetTableName(model)

	var conditions strings.Builder
	params := make(map[string]any)
	i := 0
	for field, value := range where {
		if i > 0 {
			conditions.WriteString(" AND ")
		}
		fmt.Fprintf(&conditions, "%s = $w%d", field, i)
		params[fmt.Sprintf("w%d", i)] = value
		i++
	}

	sql := "SELECT * FROM " + table + " WHERE " + conditions.String() + " LIMIT 1"
	resp, err := surrealdb.Query[[]map[string]any](ctx, db.raw, sql, params)
	if err != nil {
		return err
	}

	if resp != nil && len(*resp) > 0 && (*resp)[0].Result != nil && len((*resp)[0].Result) > 0 {
		content, _ := structToMap(model)
		maps.Copy(content, update)
		updateSQL := "UPDATE " + table + " SET " + formatUpdateSet(content) + " WHERE " + conditions.String()
		_, err = surrealdb.Query[any](ctx, db.raw, updateSQL, params)
		return err
	}

	for k, v := range update {
		reflect.ValueOf(model).Elem().FieldByName(k).Set(reflect.ValueOf(v))
	}
	return Create(ctx, db, model)
}

func (db *DB) Transaction(ctx context.Context, fn func(*DB) error) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	return tx.Commit(ctx)
}

func SoftDelete[T any](ctx context.Context, db *DB, model *T) error {
	table := GetTableName(model)
	id := reflect.ValueOf(model).Elem().FieldByName("ID").Interface().(models.RecordID)

	sql := "UPDATE " + table + " SET deleted_at = time::now() WHERE id = $id"
	_, err := surrealdb.Query[any](ctx, db.raw, sql, map[string]any{"id": id.String()})
	return err
}

func Restore[T any](ctx context.Context, db *DB, model *T) error {
	table := GetTableName(model)
	id := reflect.ValueOf(model).Elem().FieldByName("ID").Interface().(models.RecordID)

	sql := "UPDATE " + table + " SET deleted_at = null WHERE id = $id"
	_, err := surrealdb.Query[any](ctx, db.raw, sql, map[string]any{"id": id.String()})
	return err
}

func ForceDelete[T any](ctx context.Context, db *DB, model *T) error {
	table := GetTableName(model)
	id := reflect.ValueOf(model).Elem().FieldByName("ID").Interface().(models.RecordID)

	sql := "DELETE FROM " + table + " WHERE id = $id"
	_, err := surrealdb.Query[any](ctx, db.raw, sql, map[string]any{"id": id.String()})
	return err
}
