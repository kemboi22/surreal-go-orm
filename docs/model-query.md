# Model Query

Use `QueryModel` for typed queries that return your Go structs directly. It works with models that embed `Model` or implement the table name interface.

## Getting Started

```go
mq := surrealgoorm.QueryModel[User](ctx, db)
```

## Fetch All

```go
var users []User
err := surrealgoorm.QueryModel[User](ctx, db).
    All(ctx, &users)
```

## Where Clauses

### Basic Where

```go
var users []User
err := surrealgoorm.QueryModel[User](ctx, db).
    Where("active", "=", true).
    All(ctx, &users)
```

### Multiple Conditions

```go
err := surrealgoorm.QueryModel[User](ctx, db).
    Where("active", "=", true).
    Where("age", ">", 18).
    All(ctx, &users)
```

### WhereIn

```go
err := surrealgoorm.QueryModel[User](ctx, db).
    WhereIn("id", []string{"users:1", "users:2"}).
    All(ctx, &users)
```

### WhereLike

```go
err := surrealgoorm.QueryModel[User](ctx, db).
    WhereLike("name", "J%").
    All(ctx, &users)
```

### WhereNull / WhereNotNull

```go
err := surrealgoorm.QueryModel[User](ctx, db).
    WhereNull("email_verified_at").
    All(ctx, &users)
```

## Ordering

```go
// Ascending
err := surrealgoorm.QueryModel[User](ctx, db).
    OrderBy("name").
    All(ctx, &users)

// Descending
err := surrealgoorm.QueryModel[User](ctx, db).
    OrderByDesc("created_at").
    All(ctx, &users)
```

## Pagination

```go
err := surrealgoorm.QueryModel[User](ctx, db).
    Limit(10).
    Offset(20).
    All(ctx, &users)
```

## Find One

```go
var user User
err := surrealgoorm.QueryModel[User](ctx, db).
    Where("email", "=", "john@example.com").
    FindOne(ctx, &user)
```

Returns `ErrNotFound` if no match.

## Find By ID

```go
var user User
err := surrealgoorm.QueryModel[User](ctx, db).
    Find(ctx, "users:john", &user)
```

Also returns `ErrNotFound` for missing records.

## First / Last

```go
// First ordered by name
var user User
err := surrealgoorm.QueryModel[User](ctx, db).
    OrderBy("name").
    First(ctx, &user)

// Last by created_at
var user User
err := surrealgoorm.QueryModel[User](ctx, db).
    OrderByDesc("created_at").
    First(ctx, &user)
```

## Count

```go
count, err := surrealgoorm.QueryModel[User](ctx, db).
    Where("active", "=", true).
    Count(ctx)
```

## Exists

```go
exists, err := surrealgoorm.QueryModel[User](ctx, db).
    Where("email", "=", "john@example.com").
    Exists(ctx)
```

## Soft Delete

`QueryModel` automatically handles soft delete for models that include `deleted_at` fields.

### Default Behavior

Models with `SoftDeletesModel` embed have deleted records filtered:

```go
var posts []Post
err := surrealgoorm.QueryModel[Post](ctx, db).All(ctx, &posts)
// SQL: SELECT * FROM posts WHERE deleted_at = NONE
```

### Include Deleted

```go
err := surrealgoorm.QueryModel[Post](ctx, db).
    WithTrashed().
    All(ctx, &posts)
```

### Only Deleted

```go
err := surrealgoorm.QueryModel[Post](ctx, db).
    OnlyTrashed().
    All(ctx, &posts)
```

### Restore

```go
err := surrealgoorm.QueryModel[Post](ctx, db).
    WithTrashed().
    Where("id", "=", "posts:123").
    Update(ctx, map[string]any{
        "deleted_at": nil,
    })
```

##Relations

Load related models in a single query.

### With

```go
var users []User
err := surrealgoorm.QueryModel[User](ctx, db).
    With("posts").
    All(ctx, &users)
```

This fetches posts for each user and populates the `Posts` field.

### Nested Relations

```go
err := surrealgoorm.QueryModel[User](ctx, db).
    With("posts.comments").
    All(ctx, &users)
```

### Preload vs Eager Load

`With` uses eager loading. For larger datasets, consider preload:

```go
// Eager load all at once
var users []User
err := surrealgoorm.QueryModel[User](ctx, db).
    With("posts").
    All(ctx, &users)

// Or preload manually
var users []User
if err := surrealgoorm.QueryModel[User](ctx, db).All(ctx, &users); err != nil {
    return err
}

postIDs := make([]string, len(users))
for i, u := range users {
    postIDs[i] = u.ID.String()
}

var posts []Post
err = surrealgoorm.QueryModel[Post](ctx, db).
    WhereIn("author_id", postIDs).
    All(ctx, &posts)
```

## Transactions

```go
err := db.Transaction(ctx, func(tx *surrealgoorm.DB) error {
    return surrealgoorm.QueryModel[User](tx, db).
        Where("id", "=", "users:123").
        Update(ctx, map[string]any{"active": false})
})
```

## Pagination Helper

```go
type Pagination struct {
    Page    int
    PerPage int
}

func Paginate[T any](ctx context.Context, db *surrealgoorm.DB, page, perPage int) ([]T, int, error) {
    offset := (page - 1) * perPage
    
    var items []T
    err := surrealgoorm.QueryModel[T](ctx, db).
        Limit(perPage).
        Offset(offset).
        All(ctx, &items)
    
    count, err := surrealgoorm.QueryModel[T](ctx, db).Count(ctx)
    
    return items, count, err
}

// Usage
users, total, err := Paginate[User](ctx, db, 1, 10)
```

## Full Examples

### User Search

```go
func SearchUsers(ctx context.Context, db *surrealgoorm.DB, q string, limit int) ([]User, error) {
    var users []User
    
    mq := surrealgoorm.QueryModel[User](ctx, db)
    
    if q != "" {
        mq = mq.WhereLike("name", "%"+q+"%")
    }
    
    err := mq.
        OrderByDesc("created_at").
        Limit(limit).
        All(ctx, &users)
    
    return users, err
}
```

### User Profile with Posts

```go
func GetUserWithPosts(ctx context.Context, db *surrealgoorm.DB, id string) (*User, error) {
    var user User
    
    err := surrealgoorm.QueryModel[User](ctx, db).
        With("posts").
        Find(ctx, id, &user)
    
    return &user, err
}
```

### Soft Deleted Posts

```go
func GetDeletedPosts(ctx context.Context, db *surrealgoorm.DB) ([]Post, error) {
    var posts []Post
    
    err := surrealgoorm.QueryModel[Post](ctx, db).
        OnlyTrashed().
        OrderByDesc("deleted_at").
        All(ctx, &posts)
    
    return posts, err
}
```

## Error Handling

All methods return Go errors:

```go
err := surrealgoorm.QueryModel[User](ctx, db).
    Find(ctx, "users:john", &user)
if err != nil {
    if errors.Is(err, surrealgoorm.ErrNotFound) {
        // handle not found
    }
    return err
}
```

Common errors:
- `ErrNotFound` - record not found
- Connection errors
- Type conversion errors