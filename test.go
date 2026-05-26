package surrealgoorm

import (
	"context"
	"log"
)

func Up(schema Schema) {
	err := schema.CreateTable(context.Background(), "users", func(t *Table) {
		t.String("name")
		t.String("email").Unique()
	})
	if err != nil {
		log.Print(err)
	}
}
