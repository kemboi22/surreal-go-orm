package surrealgoorm

func (m Model[T]) WithTrashed() *Model[T] {
	m.state.trash = trashWith
	return &m
}

func (m Model[T]) OnlyTrashed() *Model[T] {
	m.state.trash = trashOnly
	return &m
}
