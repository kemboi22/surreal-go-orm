package surrealgoorm

import "errors"

var (
	ErrNotFound           = errors.New("record not found")
	ErrEmptyTableName     = errors.New("table name is empty")
	ErrUnsafeMutation     = errors.New("unsafe mutation blocked")
	ErrMissingWhereClause = errors.New("missing where clause")
	ErrInvalidModel       = errors.New("invalid model")
	ErrNoActiveTx         = errors.New("no active transaction")
)

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}
