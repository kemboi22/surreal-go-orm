package surrealgoorm

// WhereContains filters records whose array/set field contains value.
//
//	WHERE tags CONTAINS 'vip'
func (m Model[T]) WhereContains(column string, value any) *Model[T] {
	m.sq.Where(column+" CONTAINS ?", value)
	m.state.conds = append(m.state.conds, condition{sql: column + " CONTAINS ?", args: []any{value}})
	return &m
}

// WhereContainsAny filters records whose array/set field contains at least one
// of the supplied values.
//
//	WHERE tags CONTAINSANY ['vip', 'lead']
func (m Model[T]) WhereContainsAny(column string, values ...any) *Model[T] {
	if len(values) == 0 {
		return &m
	}
	m.sq.Where(column+" CONTAINSANY ?", values)
	m.state.conds = append(m.state.conds, condition{sql: column + " CONTAINSANY ?", args: []any{values}})
	return &m
}
