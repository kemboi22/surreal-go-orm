# Getting Started

## Install

```bash
go get github.com/kemboi22/surreal-go-orm
```

## Connect

```go
dbName := "app"
nsName := "app"

db, err := surrealgoorm.Connect(ctx, surrealgoorm.Config{
	URL:       "ws://localhost:8000",
	Username:  "root",
	Password:  "root",
	Namespace: &nsName,
	Database:  &dbName,
})
if err != nil {
	log.Fatal(err)
}
defer db.Close(ctx)
```

## First Model

```go
type User struct {
	surrealgoorm.Model
	Name  string `orm:"column:name"`
	Email string `orm:"column:email"`
}

func (User) TableName() string {
	return "users"
}
```

## First Query

```go
var users []User

err := surrealgoorm.QueryModel[User](ctx, db).
	Where("email", "=", "john@example.com").
	All(ctx, &users)
```
