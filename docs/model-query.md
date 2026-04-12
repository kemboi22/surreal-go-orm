# Model Query

Use this when you want typed results.

```go
mq := surrealgoorm.QueryModel[User](ctx, db)
```

## Read

```go
var users []User
err := surrealgoorm.QueryModel[User](ctx, db).
	Where("active", "=", true).
	OrderBy("name").
	All(ctx, &users)
```

## Find One

```go
var user User
err := surrealgoorm.QueryModel[User](ctx, db).
	Where("email", "=", "john@example.com").
	FindOne(ctx, &user)
```

## Find By Id

```go
var user User
err := surrealgoorm.QueryModel[User](ctx, db).
	Find(ctx, "users:john", &user)
```

## Soft Delete Scope

`ModelQuery[T]` auto-applies soft delete only when `T` has `deleted_at`.

```go
var posts []Post
err := surrealgoorm.QueryModel[Post](ctx, db).All(ctx, &posts)
```

To change the scope:

```go
surrealgoorm.QueryModel[Post](ctx, db).WithTrashed()
surrealgoorm.QueryModel[Post](ctx, db).OnlyTrashed()
```

## Relations

```go
var users []User
err := surrealgoorm.QueryModel[User](ctx, db).
	With("posts").
	All(ctx, &users)
```
