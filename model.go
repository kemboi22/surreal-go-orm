package surrealgoorm

import (
	"time"

	"github.com/surrealdb/surrealdb.go/pkg/models"
)

type Model struct {
	ID        models.RecordID `orm:"column:id;primary"`
	CreatedAt time.Time       `orm:"column:created_at;auto"`
	UpdatedAt time.Time       `orm:"column:updated_at;auto"`
	Active    bool            `orm:"column:active"`
}

func (m *Model) TableName() string {
	return ""
}
func (m *Model) PrimaryKey() string {
	return "id"
}

type Timestamps struct {
	CreatedAt time.Time `orm:"column:created_at;auto"`
	UpdatedAt time.Time `orm:"column:updated_at;auto"`
}
type SoftDeletes struct {
	DeletedAt *time.Time `orm:"column:deleted_at"`
}
type TimestampsModel struct {
	Timestamps
}

func (m *TimestampsModel) TableName() string {
	return ""
}

type SoftDeletesModel struct {
	SoftDeletes
	Timestamps
}

func (m *SoftDeletesModel) TableName() string {
	return ""
}

type TableName interface {
	TableName() string
}

type PrimaryKey interface {
	PrimaryKey() string
}
