# CRUD

## Create

```go
user := &User{
	Name:  "John",
	Email: "john@example.com",
}

err := surrealgoorm.Create(ctx, db, user)
```

## Find

```go
var user User
err := surrealgoorm.Find(ctx, db, "users:john", &user)
```

## All

```go
var users []User
err := surrealgoorm.All(ctx, db, &users)
```

## Update

```go
user.Name = "Jane"
err := surrealgoorm.Update(ctx, db, user)
```

## Delete

`Delete()` removes the record by id.

```go
err := surrealgoorm.Delete(ctx, db, user)
```

If you want soft delete, use a query builder with `SoftDeletes()` or use a model with `deleted_at` and call the soft delete helpers.
