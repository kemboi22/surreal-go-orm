package surrealgoorm

import "context"

type Pagination[T any] struct {
	Data        []T   `json:"data"`
	Total       int64 `json:"total"`
	PerPage     int   `json:"per_page"`
	CurrentPage int   `json:"current_page"`
	LastPage    int   `json:"last_page"`
	From        int   `json:"from"`
	To          int   `json:"to"`
}

func (m Model[T]) Paginate(ctx context.Context, page, perPage int) (*Pagination[T], error) {
	if m.state.err != nil {
		return nil, m.state.err
	}
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 15
	}
	total, err := m.Count(ctx)
	if err != nil {
		return nil, err
	}
	q := m.sq.Start((page - 1) * perPage).Limit(perPage)
	sql, vars := q.Build()
	sql = m.applyTrash(sql)
	rows, err := m.queryAll(ctx, sql, vars)
	if err != nil {
		return nil, err
	}
	if err := m.eagerLoad(ctx, rows); err != nil {
		return nil, err
	}
	lastPage := int((int64(total) + int64(perPage) - 1) / int64(perPage))
	if lastPage < 1 {
		lastPage = 1
	}
	from := 0
	to := 0
	if len(rows) > 0 {
		from = (page-1)*perPage + 1
		to = from + len(rows) - 1
	}
	return &Pagination[T]{
		Data:        rows,
		Total:       int64(total),
		PerPage:     perPage,
		CurrentPage: page,
		LastPage:    lastPage,
		From:        from,
		To:          to,
	}, nil
}
