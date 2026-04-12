# Surreal Go ORM Docs

Small ORM for SurrealDB in Go.

## Install

```bash
go get github.com/kemboi22/surreal-go-orm
```

## Docs

- [Getting Started](./getting-started.md)
- [Configuration](./configuration.md)
- [Models](./models.md)
- [CRUD](./crud.md)
- [Query Builder](./query-builder.md)
- [Model Query](./model-query.md)
- [Relations](./relations.md)
- [Implementation Notes](../implementation.md)
- [Learn Notes](../learn.md)

## Quick Example

```go
type User struct {
	surrealgoorm.Model
	Name  string `orm:"column:name"`
	Email string `orm:"column:email"`
}

func (User) TableName() string {
	return "users"
}

ctx := context.Background()

db, err := surrealgoorm.Connect(ctx, surrealgoorm.Config{
	URL:      "ws://localhost:8000",
	Username: "root",
	Password: "root",
})
if err != nil {
	log.Fatal(err)
}
defer db.Close(ctx)

var users []User
err = surrealgoorm.QueryModel[User](ctx, db).
	Where("name", "=", "John").
	All(ctx, &users)
```
