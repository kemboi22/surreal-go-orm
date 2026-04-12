package surrealgoorm

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"strings"
	"time"

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
	selectRaws   []string
	wheres       []WhereCondition
	orders       []OrderBy
	orderRaws    []string
	limitVal     int
	offsetVal    int
	withs        []string
	groupBys     []string
	softDelete   bool
	onlyTrashed  bool
	withTrashed  bool
	allowAll     bool
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

func (qb *QueryBuilder) SelectRaw(sql string) *QueryBuilder {
	qb.selectRaws = append(qb.selectRaws, sql)
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

func (qb *QueryBuilder) WhereLike(field string, value string) *QueryBuilder {
	qb.wheres = append(qb.wheres, WhereCondition{Field: field, Operator: "CONTAINS", Value: value})
	return qb
}

func (qb *QueryBuilder) WhereExists(subquery string, bindings ...any) *QueryBuilder {
	qb.wheres = append(qb.wheres, WhereCondition{
		Field: "(" + "EXISTS(" + formatRawCondition(subquery, len(qb.wheres), bindings) + ")" + ")",
	})
	return qb
}

func (qb *QueryBuilder) WhereGroup(fn func(*QueryBuilder)) *QueryBuilder {
	group := &QueryBuilder{}
	fn(group)
	groupSQL := buildWhereClause(group.wheres, true)
	groupBindings := flattenBindings(group.wheres)
	qb.wheres = append(qb.wheres, WhereCondition{
		Field: "(" + strings.ReplaceAll(groupSQL, "$w", "?") + ")",
		Value: groupBindings,
	})
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

func (qb *QueryBuilder) OrderByRaw(sql string) *QueryBuilder {
	qb.orderRaws = append(qb.orderRaws, sql)
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

func (qb *QueryBuilder) AllowAll() *QueryBuilder {
	qb.allowAll = true
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

func (qb *QueryBuilder) SoftDeletes() *QueryBuilder {
	qb.softDelete = true
	return qb
}

func (qb *QueryBuilder) WithTrashed() *QueryBuilder {
	qb.softDelete = true
	qb.withTrashed = true
	return qb
}

func (qb *QueryBuilder) OnlyTrashed() *QueryBuilder {
	qb.softDelete = true
	qb.onlyTrashed = true
	return qb
}

func (qb *QueryBuilder) buildSQL() string {
	var sql strings.Builder
	sql.WriteString("SELECT ")
	selectParts := append([]string{}, qb.selectFields...)
	selectParts = append(selectParts, qb.selectRaws...)
	if len(selectParts) > 0 {
		sql.WriteString(formatSelects(selectParts))
	} else {
		sql.WriteString("*")
	}
	sql.WriteString(" FROM " + qb.table)

	if len(qb.wheres) > 0 {
		sql.WriteString(" WHERE ")
		sql.WriteString(buildWhereClause(qb.wheres, true))
	}

	appendSoftDeleteClause(&sql, len(qb.wheres) > 0, qb.softDelete, qb.withTrashed, qb.onlyTrashed)

	if len(qb.withs) > 0 {
		sql.WriteString(" FETCH " + formatSelects(qb.withs))
	}

	if len(qb.groupBys) > 0 {
		sql.WriteString(" GROUP BY " + formatSelects(qb.groupBys))
	}

	if len(qb.orders) > 0 || len(qb.orderRaws) > 0 {
		orderParts := make([]string, 0, len(qb.orders)+len(qb.orderRaws))
		for _, o := range qb.orders {
			part := o.Field
			if o.Desc {
				part += " DESC"
			}
			orderParts = append(orderParts, part)
		}
		orderParts = append(orderParts, qb.orderRaws...)
		sql.WriteString(" ORDER BY " + strings.Join(orderParts, ", "))
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
	return buildWhereParams(qb.wheres)
}

func (qb *QueryBuilder) SQL() (string, map[string]any) {
	return qb.buildSQL(), qb.buildParams()
}

func (qb *QueryBuilder) All(ctx context.Context, results any) error {
	if err := ensureTableName(qb.table); err != nil {
		return err
	}
	sql, params := qb.buildSQL(), qb.buildParams()
	resp, err := qb.db.query(ctx, sql, params)
	if err != nil {
		return err
	}
	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil || len((*resp)[0].Result) == 0 {
		return nil
	}
	return mapSliceToStruct((*resp)[0].Result, results)
}

func (qb *QueryBuilder) One(ctx context.Context, result any) error {
	if err := ensureTableName(qb.table); err != nil {
		return err
	}
	qb.limitVal = 1
	sql, params := qb.buildSQL(), qb.buildParams()
	resp, err := qb.db.query(ctx, sql, params)
	if err != nil {
		return err
	}
	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil || len((*resp)[0].Result) == 0 {
		return ErrNotFound
	}
	return mapToStruct((*resp)[0].Result[0], result)
}

func (qb *QueryBuilder) First(ctx context.Context, result any) error {
	return qb.Limit(1).One(ctx, result)
}

func (qb *QueryBuilder) Count(ctx context.Context) (int, error) {
	if err := ensureTableName(qb.table); err != nil {
		return 0, err
	}
	sql, params := qb.buildSQL(), qb.buildParams()
	countSQL := "SELECT count() as count FROM (" + sql + ") as subquery"
	resp, err := qb.db.query(ctx, countSQL, params)
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
	resp, err := qb.db.query(ctx, sumSQL, params)
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
	resp, err := qb.db.query(ctx, avgSQL, params)
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
	resp, err := qb.db.query(ctx, minSQL, params)
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
	resp, err := qb.db.query(ctx, maxSQL, params)
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
	if err := ensureTableName(qb.table); err != nil {
		return err
	}
	var content map[string]any
	switch v := data.(type) {
	case map[string]any:
		filtered := make(map[string]any)
		for k, val := range v {
			if pt, ok := val.(*time.Time); ok && pt == nil {
				continue
			}
			if val != nil {
				filtered[k] = val
			}
		}
		content = filtered
	default:
		var err error
		content, err = structToMap(data)
		if err != nil {
			return err
		}
		filtered := make(map[string]any)
		for k, val := range content {
			if pt, ok := val.(*time.Time); ok && pt == nil {
				continue
			}
			if val != nil {
				filtered[k] = val
			}
		}
		content = filtered
	}

	fields := make([]string, 0, len(content))
	values := make(map[string]any, len(content))
	for k, v := range content {
		fields = append(fields, k)
		values[k] = v
	}

	if len(fields) == 0 {
		return nil
	}
	sql := "CREATE " + qb.table + " CONTENT $data"
	_, err := qb.db.execQuery(ctx, sql, map[string]any{"data": values})
	return err
}

func (qb *QueryBuilder) Update(ctx context.Context, data any) error {
	if err := ensureTableName(qb.table); err != nil {
		return err
	}
	if err := ensureSafeMutation(qb.allowAll, qb.wheres); err != nil {
		return err
	}
	content, err := updateContentMap(data)
	if err != nil {
		return err
	}
	setSQL, setParams := buildUpdateSetClause(content)
	params := mergeParams(qb.buildParams(), setParams)
	sql := "UPDATE " + qb.table + " SET " + setSQL + " WHERE " + extractWhereClause(qb.wheres, true)
	_, err = qb.db.execQuery(ctx, sql, params)
	return err
}

func (qb *QueryBuilder) Delete(ctx context.Context) error {
	if err := ensureTableName(qb.table); err != nil {
		return err
	}
	if err := ensureSafeMutation(qb.allowAll, qb.wheres); err != nil {
		return err
	}
	params := qb.buildParams()
	if qb.softDelete {
		sql := "UPDATE " + qb.table + " SET deleted_at = time::now() WHERE " + extractWhereClause(qb.wheres, true)
		_, err := qb.db.execQuery(ctx, sql, params)
		return err
	}
	sql := "DELETE FROM " + qb.table + " WHERE " + extractWhereClause(qb.wheres, true)
	_, err := qb.db.execQuery(ctx, sql, params)
	return err
}

func (qb *QueryBuilder) ForceDelete(ctx context.Context) error {
	if err := ensureTableName(qb.table); err != nil {
		return err
	}
	if err := ensureSafeMutation(qb.allowAll, qb.wheres); err != nil {
		return err
	}
	oldWithTrashed := qb.withTrashed
	oldOnlyTrashed := qb.onlyTrashed
	qb.withTrashed = true
	qb.onlyTrashed = true
	params := qb.buildParams()
	sql := "DELETE FROM " + qb.table + " WHERE " + extractWhereClause(qb.wheres, true)
	_, err := qb.db.execQuery(ctx, sql, params)
	qb.withTrashed = oldWithTrashed
	qb.onlyTrashed = oldOnlyTrashed
	return err
}

func (qb *QueryBuilder) Restore(ctx context.Context) error {
	if err := ensureTableName(qb.table); err != nil {
		return err
	}
	if err := ensureSafeMutation(qb.allowAll, qb.wheres); err != nil {
		return err
	}
	params := qb.buildParams()
	sql := "UPDATE " + qb.table + " SET deleted_at = NONE WHERE " + extractWhereClause(qb.wheres, true)
	_, err := qb.db.execQuery(ctx, sql, params)
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
	resp, err := qb.db.query(ctx, sql, params)
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

func (qb *QueryBuilder) InsertMany(ctx context.Context, rows []map[string]any) error {
	for _, row := range rows {
		if err := qb.Insert(ctx, row); err != nil {
			return err
		}
	}
	return nil
}

func (qb *QueryBuilder) UpdateMany(ctx context.Context, rows []map[string]any) error {
	for _, row := range rows {
		if err := qb.Update(ctx, row); err != nil {
			return err
		}
	}
	return nil
}

func (qb *QueryBuilder) FirstOrFail(ctx context.Context, result any) error {
	return qb.First(ctx, result)
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
	resp, err := db.query(ctx, sql, params)
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
	resp, err := db.query(ctx, sql, params)
	if err != nil {
		return err
	}

	if resp != nil && len(*resp) > 0 && (*resp)[0].Result != nil && len((*resp)[0].Result) > 0 {
		content, _ := structToMap(model)
		maps.Copy(content, update)
		setSQL, setParams := buildUpdateSetClause(content)
		updateSQL := "UPDATE " + table + " SET " + setSQL + " WHERE " + conditions.String()
		_, err = db.execQuery(ctx, updateSQL, mergeParams(params, setParams))
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
	_, err := db.execQuery(ctx, sql, map[string]any{"id": id.String()})
	return err
}

func Restore[T any](ctx context.Context, db *DB, model *T) error {
	table := GetTableName(model)
	id := reflect.ValueOf(model).Elem().FieldByName("ID").Interface().(models.RecordID)

	sql := "UPDATE " + table + " SET deleted_at = NONE WHERE id = $id"
	_, err := db.execQuery(ctx, sql, map[string]any{"id": id.String()})
	return err
}

func ForceDelete[T any](ctx context.Context, db *DB, model *T) error {
	table := GetTableName(model)
	id := reflect.ValueOf(model).Elem().FieldByName("ID").Interface().(models.RecordID)

	sql := "DELETE FROM " + table + " WHERE id = $id"
	_, err := db.execQuery(ctx, sql, map[string]any{"id": id.String()})
	return err
}
