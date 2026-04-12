# CRUD

Surreal Go ORM provides simple CRUD operations for your models. These work with any struct that embeds `Model`.

## Create

Create a new record in the database.

```go
user := &User{
    Name:  "John",
    Email: "john@example.com",
}

err := surrealgoorm.Create(ctx, db, user)
```

### With Custom ID

Set a custom record ID:

```go
import "github.com/surrealdb/surrealdb.go/pkg/models"

user := &User{
    Model: surrealgoorm.Model{
        ID: models.NewRecordID("users", "john"),
    },
    Name:  "John",
    Email: "john@example.com",
}

err := surrealgoorm.Create(ctx, db, user)
```

### Create Many

Insert multiple records at once:

```go
users := []*User{
    {Name: "John", Email: "john@example.com"},
    {Name: "Jane", Email: "jane@example.com"},
}

for _, user := range users {
    if err := surrealgoorm.Create(ctx, db, user); err != nil {
        return err
    }
}
```

## Find

Find a single record by its ID.

```go
var user User
err := surrealgoorm.Find(ctx, db, "users:john", &user)
```

Returns `ErrNotFound` if the record doesn't exist.

```go
var user User
err := surrealgoorm.Find(ctx, db, "users:john", &user)
if err != nil {
    if errors.Is(err, surrealgoorm.ErrNotFound) {
        // record not found
    }
    return err
}
```

## All

Fetch all records of a type.

```go
var users []User
err := surrealgoorm.All(ctx, db, &users)
```

This loads all records from the table. For large tables, use `QueryModel` with pagination instead.

## Update

Update an existing record. The record must have an ID set.

```go
user.Name = "Jane"
err := surrealgoorm.Update(ctx, db, user)
```

The ORM updates only the fields that differ from their zero values.

### UpdateSpecific Fields

Update specific fields using the query builder:

```go
err := db.Query("users").
    Where("id", "=", "users:john").
    Update(ctx, map[string]any{
        "name": "Jane",
    })
```

## Delete

Hard delete removes the record permanently.

```go
err := surrealgoorm.Delete(ctx, db, user)
```

### Soft Delete

For soft delete, use a model with `SoftDeletesModel`:

```go
type Post struct {
    surrealgoorm.Model
    Title string `orm:"column:title"`
    surrealgoorm.SoftDeletesModel
}
```

Then use the query builder:

```go
err := db.Query("posts").
    SoftDeletes().
    Where("id", "=", "posts:123").
    Delete(ctx)
```

This sets `deleted_at` instead of removing the record.

### Restore Soft Deleted

Restore a soft deleted record:

```go
err := db.Query("posts").
    WithTrashed().
    Where("id", "=", "posts:123").
    Update(ctx, map[string]any{
        "deleted_at": nil,
    })
```

## FindOrCreate

Find an existing record or create a new one:

```go
user := &User{Email: "john@example.com"}
err := surrealgoorm.Find(ctx, db, "users:john", user)
if errors.Is(err, surrealgoorm.ErrNotFound) {
    user.Name = "John"
    err = surrealgoorm.Create(ctx, db, user)
}
```

## Upsert

Create or update in one call:

```go
user := &User{
    Model:   surrealgoorm.Model{ID: models.NewRecordID("users", "john")},
    Name:    "John",
    Email:   "john@example.com",
    Age:     30,
}

// Try update first
err := surrealgoorm.Update(ctx, db, user)
if errors.Is(err, surrealgoorm.ErrNotFound) {
    // Not found, create instead
    err = surrealgoorm.Create(ctx, db, user)
}
```

## Error Types

The CRUD functions return these errors:

- `ErrNotFound` - record doesn't exist
- `ErrInvalidID` - invalid record ID
- `ErrNilModel` - nil model pointer

```go
import "errors"

var user User
err := surrealgoorm.Find(ctx, db, "users:john", &user)
if errors.Is(err, surrealgoorm.ErrNotFound) {
    // handle
}
```

## Transactions

Wrap CRUD operations in a transaction:

```go
err := db.Transaction(ctx, func(tx *surrealgoorm.DB) error {
    user := &User{Name: "John"}
    if err := surrealgoorm.Create(ctx, tx, user); err != nil {
        return err
    }

    post := &Post{AuthorID: user.ID.String(), Title: "Hello"}
    return surrealgoorm.Create(ctx, tx, post)
})
```