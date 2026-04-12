# Query Builder

Use the query builder when you want to query tables directly by name. It's useful for dynamic queries and raw SQL operations.

## Getting Started

```go
qb := db.Query("posts")
```

## Select

Fetch results with `All`:

```go
var posts []Post
err := db.Query("posts").
    All(ctx, &posts)
```

Or as maps:

```go
var results []map[string]any
err := db.Query("posts").
    All(ctx, &results)
```

### Select Specific Columns

```go
var results []map[string]any
err := db.Query("posts").
    Select("id", "title", "created_at").
    All(ctx, &results)
```

## Where Clauses

### Basic Where

```go
var posts []Post
err := db.Query("posts").
    Where("published", "=", true).
    All(ctx, &posts)
```

### Multiple Conditions

```go
err := db.Query("posts").
    Where("published", "=", true).
    Where("author_id", "=", "users:123").
    All(ctx, &posts)
```

### Where In

```go
err := db.Query("posts").
    WhereIn("id", []string{"posts:1", "posts:2", "posts:3"}).
    All(ctx, &posts)
```

### Where Like

Pattern matching:

```go
err := db.Query("posts").
    WhereLike("title", "Go%").
    All(ctx, &posts)
```

### Where Null / Not Null

```go
// Find posts without author
err := db.Query("posts").
    WhereNull("author_id").
    All(ctx, &posts)

// Find posts with author
err := db.Query("posts").
    WhereNotNull("author_id").
    All(ctx, &posts)
```

### Where Between

```go
err := db.Query("posts").
    WhereBetween("created_at", startTime, endTime).
    All(ctx, &posts)
```

## Ordering

```go
// Ascending
err := db.Query("posts").
    OrderBy("created_at").
    All(ctx, &posts)

// Descending
err := db.Query("posts").
    OrderByDesc("created_at").
    All(ctx, &posts)

// Multiple
err := db.Query("posts").
    OrderBy("author_id").
    OrderByDesc("created_at").
    All(ctx, &posts)
```

## Pagination

```go
// Limit
err := db.Query("posts").
    Limit(10).
    All(ctx, &posts)

// Offset
err := db.Query("posts").
    Offset(20).
    Limit(10).
    All(ctx, &posts)

// Start / Limit (alternative)
err := db.Query("posts").
    Start(0).
    Limit(10).
    All(ctx, &posts)
```

## Count

```go
count, err := db.Query("posts").
    Where("published", "=", true).
    Count(ctx)
```

## First / Find One

Get the first result:

```go
var post Post
err := db.Query("posts").
    Where("published", "=", true).
    OrderByDesc("created_at").
    First(ctx, &post)
```

## Insert

### Insert One

```go
err := db.Query("posts").
    Insert(ctx, map[string]any{
        "title":   "Hello World",
        "content": "First post",
    })
```

### Insert Many

```go
posts := []map[string]any{
    {"title": "Post 1", "content": "Content 1"},
    {"title": "Post 2", "content": "Content 2"},
    {"title": "Post 3", "content": "Content 3"},
}

err := db.Query("posts").
    InsertMany(ctx, posts)
```

## Update

### Update One

```go
err := db.Query("posts").
    Where("id", "=", "posts:123").
    Update(ctx, map[string]any{
        "title": "Updated Title",
    })
```

### Update All

```go
err := db.Query("posts").
    Where("published", "=", false).
    Update(ctx, map[string]any{
        "published": true,
    })
```

### Increment / Decrement

```go
// Increment a counter
err := db.Query("posts").
    Where("id", "=", "posts:123").
    Increment("views", 1)

// Decrement
err := db.Query("posts").
    Where("id", "=", "posts:123").
    Decrement("stock", 1)
```

## Delete

### Hard Delete

Permanently remove records:

```go
err := db.Query("posts").
    Where("id", "=", "posts:123").
    Delete(ctx)

// Delete all
err := db.Query("posts").
    AllowAll().
    Delete(ctx)
```

### Soft Delete

For tables with `deleted_at` field:

```go
err := db.Query("posts").
    SoftDeletes().
    Where("id", "=", "posts:123").
    Delete(ctx)
```

This sets `deleted_at` to current time instead of removing the record.

## Soft Delete Scopes

By default, soft deleted records are excluded. Change this behavior:

```go
// Include soft deleted
db.Query("posts").WithTrashed()

// Only soft deleted
db.Query("posts").OnlyTrashed()

// Exclude soft deleted (default)
db.Query("posts").SoftDeletes()
```

## Joins

### Inner Join

```go
var results []map[string]any
err := db.Query("posts").
    Join("authors", "posts.author_id", "authors.id").
    All(ctx, &results)
```

### Left Join

```go
err := db.Query("posts").
    LeftJoin("authors", "posts.author_id", "authors.id").
    All(ctx, &results)
```

### Custom Join

```go
err := db.Query("posts").
    JoinRaw("JOIN authors ON posts.author_id = authors.id AND authors.active = true").
    All(ctx, &results)
```

## Group By

```go
err := db.Query("posts").
    GroupBy("author_id").
    All(ctx, &results)
```

## Aggregate

After grouping:

```go
// Count per group
err := db.Query("posts").
    GroupBy("author_id").
    Count(ctx, "post_count")

// Sum
err := db.Query("orders").
    GroupBy("user_id").
    Sum("total_amount")

// Avg, Min, Max
err := db.Query("orders").
    GroupBy("user_id").
    Avg("total_amount")
```

## Raw SQL

### Raw Query

```go
var results []map[string]any
err := db.RawQuery("SELECT * FROM posts WHERE published = true").All(ctx, &results)
```

### Raw Exec

```go
err := db.Exec(ctx, "DELETE FROM posts WHERE deleted_at < $date", map[string]any{
    "date": fiveDaysAgo,
})
```

## Chaining Examples

```go
// Complex query
err := db.Query("posts").
    Where("published", "=", true).
    WhereLike("title", "%Go%").
    OrderByDesc("created_at").
    Limit(20).
    All(ctx, &posts)

// With count
count, err := db.Query("posts").
    Where("published", "=", true).
    Count(ctx)

// Update with conditions
err := db.Query("posts").
    Where("author_id", "=", "users:123").
    WhereNull("published_at").
    Update(ctx, map[string]any{
        "published_at": time.Now(),
    })
```

## Error Handling

Query builder methods return standard Go errors:

```go
err := db.Query("posts").All(ctx, &posts)
if err != nil {
    return fmt.Errorf("query failed: %w", err)
}
```

Common errors:
- `ErrNotFound` - no results (for `First`/`FindOne`)
- Connection errors
- SQL syntax errors