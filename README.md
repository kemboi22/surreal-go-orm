# surreal-go-orm

A lightweight, type-safe ORM for [SurrealDB](https://surrealdb.com/) built on top of the official Go client (`github.com/surrealdb/surrealdb.go`).

## Installation

```bash
go get github.com/kemboi22/surreal-go-orm
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"

    "github.com/surrealdb/surrealdb.go"
    surrealgoorm "github.com/kemboi22/surreal-go-orm"
)

type User struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func main() {
    db, err := surrealdb.New("ws://localhost:8000/rpc")
    if err != nil {
        panic(err)
    }
    defer db.Close()

    _, err = db.SignIn(&surrealdb.Auth{User: "root", Pass: "root"})
    if err != nil {
        panic(err)
    }

    _, err = db.Use("namespace", "database")
    if err != nil {
        panic(err)
    }

    ctx := context.Background()

    user, err := surrealgoorm.Query[User](db, "user").
        WhereEq("email", "alice@example.com").
        First(ctx)
    if err != nil {
        panic(err)
    }
    fmt.Println(user)
}
```

## Query Builder

The core feature is a fluent, generic query builder for SELECT queries.

### Constructor

```go
surrealgoorm.Query[T](db, "table_name")
```

Returns a `QueryBuilder[T]` for the given table, where `T` is your Go struct type.

### Methods

All builder methods return `QueryBuilder[T]` for method chaining.

| Method | Description |
|--------|-------------|
| `Select(columns ...string)` | Select specific fields (default: all) |
| `Where(column, operator string, value any)` | Add a WHERE clause (e.g. `"age", ">", 18`) |
| `WhereEq(column string, value any)` | Shorthand for equality WHERE |
| `WhereNotNull(column string)` | WHERE field IS NOT NULL |
| `WhereNull(column string)` | WHERE field IS NULL |
| `OrderBy(column string)` | Add ORDER BY |
| `Limit(limit int)` | Add LIMIT |
| `With(relations ...string)` | Add FETCH clause for eager-loading related records |
| `ToSQL()` | Returns the generated SurrealQL string with parameters inlined (useful for debugging) |
| `ToBuildSql` | Returns SQL and Vars for surrealql |
| `First(ctx)` | Executes query, returns `*T` (nil if not found) |
| `Get(ctx)` | Executes query, returns `*[]T` |

### Examples

```go
// Get all active users over 18
users, err := surrealgoorm.Query[User](db, "user").
    Where("age", ">", 18).
    WhereEq("is_active", true).
    OrderBy("name").
    Limit(10).
    Get(ctx)

// Get first match with related records (FETCH)
user, err := surrealgoorm.Query[User](db, "user").
    WhereEq("email", "bob@test.com").
    With("posts", "profile").
    First(ctx)

// Debug generated SQL (parameters inlined)
sql := surrealgoorm.Query[User](db, "user").
    WhereEq("email", "test@test.com").
    ToSQL()
fmt.Println(sql) // SELECT * FROM user WHERE email = 'test@test.com'
```

## Schema Builder

Define SurrealDB tables and columns using a DSL.

### Creating a Table

```go
schema := surrealgoorm.Schema{db: db}

err := schema.CreateTable(context.Background(), "user", func(t *surrealgoorm.Table) {
    t.ID()                          // record<user>
    t.String("name")
    t.String("email").Unique()
    t.Int("age").Default(18).Nullable()
    t.Bool("is_active").Default(true)
    t.Float("score")
    t.Timestamps()                  // adds created_at + updated_at
})
```

### Column Types

| Method | SurrealQL Type |
|--------|---------------|
| `ID()` | `record<table>` |
| `String(name)` | `string` |
| `Int(name)` | `int` |
| `Bool(name)` | `boolean` |
| `Float(name)` | `float` |
| `Number(name)` | `number` |
| `Decimal(name)` | `decimal` |
| `Datetime(name)` | `datetime` |
| `Duration(name)` | `duration` |
| `Bytes(name)` | `bytes` |
| `Array(name)` | `array` |
| `Object(name)` | `object` |
| `Regex(name)` | `regex` |
| `Geometry(name, typ)` | `geometry<typ>` |
| `Timestamps()` | Shorthand for `created_at` + `updated_at` datetime |

### Column Modifiers

| Method | Description |
|--------|-------------|
| `.Unique()` | Adds a UNIQUE index |
| `.Default(v)` | Sets a default value |
| `.Nullable()` | Makes the field FLEXIBLE (nullable) |

### Generating SurrealQL

```go
var table surrealgoorm.Table
// ... define columns ...
sql := table.Build()
fmt.Println(sql)
// DEFINE TABLE user SCHEMAFULL
// DEFINE FIELD name ON user type string;
// DEFINE FIELD email ON user type string;
// DEFINE INDEX email_unique ON user FIELDS email UNIQUE;
```

**Note:** `CreateTable` builds the table in memory but does not execute SQL against the database. Use `table.Build()` to get the SurrealQL and run it via `db.Query()`.

## Migrations

Define migrations by implementing the `Migration` interface:

```go
type CreateUserTable struct{}

func (m CreateUserTable) Name() string { return "001_create_user_table" }

func (m CreateUserTable) Up(ctx context.Context, schema surrealgoorm.Schema) error {
    return schema.CreateTable(ctx, "user", func(t *surrealgoorm.Table) {
        t.ID()
        t.String("name")
        t.String("email").Unique()
        t.Timestamps()
    })
}

func (m CreateUserTable) Down(ctx context.Context, schema surrealgoorm.Schema) error {
    _, err := surrealdb.Query[any](ctx, schema.Db, "REMOVE TABLE user;", nil)
    return err
}
```

### Auto Migration Runner

The `migrator` sub-package provides an `AutoMigrator` to run migrations sequentially:

```go
import (
    "context"
    "github.com/surrealdb/surrealdb.go"
    surrealgoorm "github.com/kemboi22/surreal-go-orm"
    "github.com/kemboi22/surreal-go-orm/migrator"
)

func main() {
    db, err := surrealdb.New("ws://localhost:8000/rpc")
    if err != nil {
        panic(err)
    }
    defer db.Close()

    m := migrator.NewAutoMigrator(db)
    err = m.AutoMigrate(context.Background(), []surrealgoorm.Migration{
        CreateUserTable{},
        CreatePostsTable{},
    })
    if err != nil {
        panic(err)
    }
}
```

`AutoMigrator` runs each migration's `Up` in order and stops at the first error.

## API Reference

### Interfaces

```go
QueryBuilder[T any] interface {
    Select(columns ...string) QueryBuilder[T]
    Where(column, operator string, value any) QueryBuilder[T]
    WhereEq(column string, value any) QueryBuilder[T]
    WhereNotNull(column string) QueryBuilder[T]
    WhereNull(column string) QueryBuilder[T]
    OrderBy(column string) QueryBuilder[T]
    Limit(limit int) QueryBuilder[T]
    With(relations ...string) QueryBuilder[T]
    ToSQL() string
    First(ctx context.Context) (*T, error)
    Get(ctx context.Context) (*[]T, error)
}
```

### Types

```go
Model[T any]        // concrete implementation of QueryBuilder[T]
Schema              // table definition builder (holds *surrealdb.DB)
Table               // table definition (Name string, Columns []*Column)
Column              // column definition (Name, Type, IsUnique, DefaultValue, Optional)
```

### Constructor

```go
func Query[T any](db *surrealdb.DB, table string) QueryBuilder[T]
```
