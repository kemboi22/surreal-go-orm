# Query Builder

Use this when you want to query a table by name.

```go
qb := db.Query("posts")
```

## Read

```go
var posts []Post
err := db.Query("posts").
	Where("published", "=", true).
	OrderByDesc("created_at").
	Limit(10).
	All(ctx, &posts)
```

## Update

```go
err := db.Query("posts").
	Where("id", "=", "posts:hello").
	Update(ctx, map[string]any{
		"title": "New Title",
	})
```

## Delete

Hard delete:

```go
err := db.Query("posts").
	Where("id", "=", "posts:hello").
	Delete(ctx)
```

Soft delete:

```go
err := db.Query("posts").
	SoftDeletes().
	Where("id", "=", "posts:hello").
	Delete(ctx)
```

## Soft Delete Scope

`QueryBuilder` does not assume soft delete.

Use one of these when the table has `deleted_at`:

```go
db.Query("posts").SoftDeletes()
db.Query("posts").WithTrashed()
db.Query("posts").OnlyTrashed()
```

## Count

```go
count, err := db.Query("posts").
	Where("published", "=", true).
	Count(ctx)
```
