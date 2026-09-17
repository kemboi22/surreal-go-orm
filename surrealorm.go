package surrealgoorm

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/contrib/surrealql"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

type QueryBuilder[T any] interface {
	Select(columns ...string) *Model[T]
	Where(column, operator string, value any) *Model[T]
	WhereEq(column string, value any) *Model[T]
	WhereNotNull(column string) *Model[T]
	WhereNull(column string) *Model[T]
	WhereIn(column string, values []any) *Model[T]
	WhereContains(column string, value any) *Model[T]
	WhereContainsAny(column string, values ...any) *Model[T]
	OrderBy(column string, direction ...string) *Model[T]
	OrderByDesc(column string) *Model[T]
	Limit(limit int) *Model[T]
	WithNames(relations ...string) *Model[T]
	WithTrashed() *Model[T]
	OnlyTrashed() *Model[T]
	ToSQL() string
	ToBuildSQL() (string, map[string]any)
	First(ctx context.Context) (*T, error)
	Get(ctx context.Context) (*[]T, error)

	Create(ctx context.Context, data *T) (*T, error)
	CreateWithID(ctx context.Context, id any, data *T) (*T, error)
	CreateMany(ctx context.Context, data []T) ([]T, error)
	CreateFromMap(ctx context.Context, attrs map[string]any) (*T, error)
	Find(ctx context.Context, id any) (*T, error)
	FindOrFail(ctx context.Context, id any) (*T, error)
	FirstOrCreate(ctx context.Context, attrs map[string]any, values ...map[string]any) (*T, error)
	FirstOrNew(ctx context.Context, attrs map[string]any, values ...map[string]any) (*T, error)
	UpdateOrCreate(ctx context.Context, attrs map[string]any, values map[string]any) (*T, error)
	Save(ctx context.Context, model *T) (*T, error)
	Update(ctx context.Context, id any, data map[string]any) (*T, error)
	UpdateWhere(ctx context.Context, data map[string]any) ([]T, error)
	Increment(ctx context.Context, id any, column string, by int64) (*T, error)
	Decrement(ctx context.Context, id any, column string, by int64) (*T, error)
	WhereIncrement(ctx context.Context, column string, by int64) ([]T, error)
	WhereDecrement(ctx context.Context, column string, by int64) ([]T, error)
	Delete(ctx context.Context, id any) error
	DeleteWhere(ctx context.Context) error
	ForceDelete(ctx context.Context, id any) error
	Restore(ctx context.Context, id any) error
	Truncate(ctx context.Context) error

	Count(ctx context.Context) (int, error)
	Exists(ctx context.Context) (bool, error)
	Sum(ctx context.Context, column string) (float64, error)
	Avg(ctx context.Context, column string) (float64, error)
	Min(ctx context.Context, column string) (float64, error)
	Max(ctx context.Context, column string) (float64, error)
	Paginate(ctx context.Context, page, perPage int) (*Pagination[T], error)

	Relate(ctx context.Context, from models.RecordID, edge string, to models.RecordID, content map[string]any) (map[string]any, error)
}

type condition struct {
	sql  string
	args []any
}

type trashMode int

const (
	trashDefault trashMode = iota
	trashWith
	trashOnly
)

type builderState struct {
	conds []condition
	trash trashMode
	with  []*relationMeta
	err   error
}

// connProvider is implemented by *Model[T] so relationship helpers can
// recover the underlying connection (*surrealdb.DB, *surrealdb.Session or
// *surrealdb.Transaction) from any QueryBuilder value.
type connProvider interface {
	conn() any
}

type Model[T any] struct {
	table  string
	sq     *surrealql.SelectQuery
	client any
	meta   *modelMeta
	state  *builderState
}

func newModel[T any](client any, table string) *Model[T] {
	return &Model[T]{
		table:  table,
		sq:     surrealql.Select(table),
		client: client,
		meta:   reflectMeta(reflect.TypeFor[T]()),
		state:  &builderState{},
	}
}

func (m Model[T]) conn() any { return m.client }

func Query[T any](db *surrealdb.DB, table string) *Model[T] {
	return newModel[T](db, table)
}

func (m Model[T]) Select(columns ...string) *Model[T] {
	for _, col := range columns {
		m.sq.Fields(col)
	}
	return &m
}

func (m Model[T]) Where(column, operator string, value any) *Model[T] {
	cond := column + " " + operator + " ?"
	m.sq.Where(cond, value)
	m.state.conds = append(m.state.conds, condition{sql: cond, args: []any{value}})
	return &m
}

func (m Model[T]) WhereEq(column string, value any) *Model[T] {
	m.sq.WhereEq(column, value)
	m.state.conds = append(m.state.conds, condition{sql: "type::field(?) = ?", args: []any{column, value}})
	return &m
}
func (m Model[T]) WhereNotNull(column string) *Model[T] {
	m.sq.WhereNotNull(column)
	m.state.conds = append(m.state.conds, condition{sql: "type::field(?) IS NOT NULL", args: []any{column}})
	return &m
}
func (m Model[T]) WhereNull(column string) *Model[T] {
	m.sq.WhereNull(column)
	m.state.conds = append(m.state.conds, condition{sql: "type::field(?) IS NULL", args: []any{column}})
	return &m
}
func (m Model[T]) WhereIn(column string, values []any) *Model[T] {
	m.sq.Where(column+" IN ?", values)
	m.state.conds = append(m.state.conds, condition{sql: column + " IN ?", args: []any{values}})
	return &m
}

// OrderBy adds an ascending ORDER BY clause. Pass "DESC" as the optional
// direction to order descending, e.g. OrderBy("created_at", "DESC").
func (m Model[T]) OrderBy(column string, direction ...string) *Model[T] {
	if len(direction) > 0 && strings.EqualFold(strings.TrimSpace(direction[0]), "DESC") {
		m.sq.OrderByDesc(column)
		return &m
	}
	m.sq.OrderBy(column)
	return &m
}

// OrderByDesc adds an ORDER BY <column> DESC clause.
func (m Model[T]) OrderByDesc(column string) *Model[T] {
	m.sq.OrderByDesc(column)
	return &m
}
func (m Model[T]) Limit(limit int) *Model[T] {
	m.sq.Limit(limit)
	return &m
}

func (m Model[T]) WithNames(relations ...string) *Model[T] {
	for _, name := range relations {
		rel, err := m.meta.relationByName(name)
		if err != nil {
			m.state.err = err
			return &m
		}
		if err := m.addRelation(rel); err != nil {
			m.state.err = err
			return &m
		}
	}
	return &m
}

// With queues an eager-loaded relation by matching the unique field on T
// whose type is R (e.g. With[[]Post]() for a Posts []Post field).
func (m *Model[T]) With[R any]() *Model[T] {
	rel, err := m.meta.relationByType(reflect.TypeFor[R]())
	if err != nil {
		m.state.err = err
		return m
	}
	if err := m.addRelation(rel); err != nil {
		m.state.err = err
	}
	return m
}

// WithField queues an eager-loaded relation by Go field name or json name,
// verifying the field type matches R.
func (m *Model[T]) WithField[R any](name string) *Model[T] {
	rel, err := m.meta.relationByName(name)
	if err != nil {
		m.state.err = err
		return m
	}
	if rel.fieldType != reflect.TypeFor[R]() {
		m.state.err = fmt.Errorf("surrealgoorm: relation %q has type %s, not %s", name, rel.fieldType, reflect.TypeFor[R]())
		return m
	}
	if err := m.addRelation(rel); err != nil {
		m.state.err = err
	}
	return m
}

func (m *Model[T]) addRelation(rel *relationMeta) error {
	if rel.kind != relFetch && (rel.table == "" || rel.fk == "") {
		return fmt.Errorf("surrealgoorm: relation %q requires table and fk tags", rel.fieldName)
	}
	for _, existing := range m.state.with {
		if existing.fieldName == rel.fieldName {
			return nil
		}
	}
	m.state.with = append(m.state.with, rel)
	if rel.kind == relFetch {
		m.sq.Fetch(rel.jsonName)
	}
	return nil
}

func (m Model[T]) ToBuildSQL() (string, map[string]any) {
	sql, vars := m.sq.Build()
	return m.applyTrash(sql), vars
}

func (m Model[T]) ToSQL() string {
	sql, vars := m.sq.Build()
	sql = m.applyTrash(sql)
	if len(vars) == 0 {
		return sql
	}
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return len(keys[i]) > len(keys[j])
	})
	replParts := make([]string, 0, len(vars)*2)
	for _, k := range keys {
		replParts = append(replParts, "$"+k, formatValue(vars[k]))
	}
	return strings.NewReplacer(replParts...).Replace(sql)
}
func (m Model[T]) First(ctx context.Context) (*T, error) {
	if m.state.err != nil {
		return nil, m.state.err
	}
	sql, vars := m.sq.Build()
	sql = m.applyTrash(sql)
	rows, err := m.queryAll(ctx, sql, vars)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	if err := m.eagerLoad(ctx, rows); err != nil {
		return nil, err
	}
	return &rows[0], nil
}

func (m Model[T]) Get(ctx context.Context) (*[]T, error) {
	if m.state.err != nil {
		return nil, m.state.err
	}
	sql, vars := m.sq.Build()
	sql = m.applyTrash(sql)
	rows, err := m.queryAll(ctx, sql, vars)
	if err != nil {
		return nil, err
	}
	if err := m.eagerLoad(ctx, rows); err != nil {
		return nil, err
	}
	return &rows, nil
}

func (m Model[T]) recordID(id any) models.RecordID {
	switch v := id.(type) {
	case models.RecordID:
		return v
	case *models.RecordID:
		return *v
	case string:
		if strings.Contains(v, ":") {
			if rid, err := models.ParseRecordID(v); err == nil {
				return *rid
			}
		}
		return models.NewRecordID(m.table, v)
	default:
		return models.NewRecordID(m.table, v)
	}
}

func (m Model[T]) applyTrash(sql string) string {
	if m.meta == nil || !m.meta.softDelete {
		return sql
	}
	switch m.state.trash {
	case trashWith:
		return sql
	case trashOnly:
		return insertWhereClause(sql, "deleted_at IS NOT NONE")
	default:
		return insertWhereClause(sql, "deleted_at IS NONE")
	}
}

func insertWhereClause(sql, condition string) string {
	fromIdx := strings.Index(sql, " FROM ")
	if fromIdx < 0 {
		return sql
	}
	rest := sql[fromIdx+len(" FROM "):]
	cut := len(rest)
	for _, kw := range []string{" WHERE ", " SPLIT ", " GROUP ", " ORDER ", " LIMIT ", " START ", " FETCH ", " PARALLEL ", " RETURN "} {
		if i := strings.Index(rest, kw); i >= 0 && i < cut {
			cut = i
		}
	}
	before := rest[:cut]
	after := rest[cut:]
	prefix := sql[:fromIdx+len(" FROM ")] + before
	if strings.HasPrefix(after, " WHERE ") {
		return prefix + " WHERE " + condition + " AND" + after[len(" WHERE"):]
	}
	return prefix + " WHERE " + condition + after
}

// runQuery dispatches a query to the underlying SurrealDB client. The same
// Model can be backed by a *surrealdb.DB, *surrealdb.Session or
// *surrealdb.Transaction, which enables the ORM to run inside transactions.
func runQuery(ctx context.Context, client any, sql string, vars map[string]any) (*[]surrealdb.QueryResult[[]map[string]any], error) {
	switch c := client.(type) {
	case *surrealdb.DB:
		return surrealdb.Query[[]map[string]any](ctx, c, sql, vars)
	case *surrealdb.Session:
		return surrealdb.Query[[]map[string]any](ctx, c, sql, vars)
	case *surrealdb.Transaction:
		return surrealdb.Query[[]map[string]any](ctx, c, sql, vars)
	default:
		return nil, fmt.Errorf("surrealgoorm: unsupported client type %T", client)
	}
}

func (m Model[T]) queryMaps(ctx context.Context, sql string, vars map[string]any) ([]map[string]any, error) {
	res, err := runQuery(ctx, m.client, sql, vars)
	if err != nil {
		return nil, err
	}
	if len(*res) == 0 {
		return nil, nil
	}
	return (*res)[0].Result, nil
}

func (m Model[T]) queryAll(ctx context.Context, sql string, vars map[string]any) ([]T, error) {
	records, err := m.queryMaps(ctx, sql, vars)
	if err != nil {
		return nil, err
	}
	out := make([]T, 0, len(records))
	for _, rec := range records {
		var item T
		if err := mapToStruct(rec, &item); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

func (m Model[T]) baseSelect() *surrealql.SelectQuery {
	q := surrealql.Select(m.table)
	for _, c := range m.state.conds {
		q = q.Where(c.sql, c.args...)
	}
	return q
}

func (m Model[T]) filterAttrs(attrs map[string]any) map[string]any {
	if len(m.meta.fillable) > 0 {
		filtered := map[string]any{}
		for k, v := range attrs {
			if m.meta.fillable[k] {
				filtered[k] = v
			}
		}
		return filtered
	}
	if len(m.meta.guarded) > 0 {
		filtered := map[string]any{}
		for k, v := range attrs {
			if !m.meta.guarded[k] {
				filtered[k] = v
			}
		}
		return filtered
	}
	return attrs
}

func (m Model[T]) whereMap(attrs map[string]any) *Model[T] {
	q := &m
	for k, v := range attrs {
		q = q.WhereEq(k, v)
	}
	return q
}

var identifierRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// validIdentifier reports whether name is a safe SurrealDB field identifier.
// Column names are interpolated into SQL (they cannot be bound parameters), so
// the atomic counter helpers reject anything that is not a plain identifier.
func validIdentifier(name string) bool {
	return identifierRe.MatchString(name)
}
