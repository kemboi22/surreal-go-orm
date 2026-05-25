package surrealgoorm

import (
	"context"
	"log"

	"github.com/surrealdb/surrealdb.go"
)

func Up() {
	schema := Schema{
		db: &surrealdb.DB{},
	}
	err := schema.CreateTable(context.Background(), "users", func(t *Table) {
		t.String("name")
		t.String("email").Unique()
	})
	if err != nil {
		log.Print(err)
	}
}
