# surreal-go-orm

A lightweight, type-safe ORM for [SurrealDB](https://surrealdb.com/) built on top of the official Go client (`github.com/surrealdb/surrealdb.go`).

Inspired by [Laravel Eloquent](https://laravel.com/docs/eloquent): fluent query builder, active-record-style CRUD, auto timestamps, soft deletes, mass assignment, observers, aggregates and pagination.

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
    ID    string `json:"id,omitempty"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func main() {
    db, err := surrealdb.New("ws://localhost:8000/rpc")
    if err != nil {
        panic(err)
    }
    ctx := context.Background()
    defer db.Close(ctx)

    _, err = db.SignIn(ctx, surrealdb.Auth{
        Username: "root",
        Password: "root",
    })
    if err != nil {
        panic(err)
    }

    _, err = db.Use(ctx, "namespace", "database")
    if err != nil {
        panic(err)
    }

    user, err := surrealgoorm.Query[User](db, "user").
        WhereEq("email", "alice@example.com").
        First(ctx)
    if err != nil {
        panic(err)
    }
    fmt.Println(user)
}
```

## Eloquent-style CRUD

Models are plain structs. The `id` field, `created_at`, `updated_at` and `deleted_at` fields are recognised by their JSON tag.

```go
type User struct {
    ID        string     `json:"id,omitempty"`
    Name      string     `json:"name"`
    Email     string     `json:"email"`
    CreatedAt time.Time  `json:"created_at"`
    UpdatedAt time.Time  `json:"updated_at"`
    DeletedAt *time.Time `json:"deleted_at"` // enables soft deletes
}
```

### Creating

```go
u := &User{Name: "Alice", Email: "alice@example.com"}
created, err := surrealgoorm.Query[User](db, "user").Create(ctx, u)
// created.ID, created.CreatedAt, created.UpdatedAt are populated

// bulk insert
users, err := surrealgoorm.Query[User](db, "user").CreateMany(ctx, []User{...})

// create from a map (respects fillable/guarded, see Mass Assignment)
u, err := surrealgoorm.Query[User](db, "user").CreateFromMap(ctx, map[string]any{
    "name": "Bob", "email": "bob@example.com",
})

// create with an explicit record id
u, err := surrealgoorm.Query[User](db, "user").CreateWithID(ctx, "bob", &User{Name: "Bob"})
```

### Reading

```go
user, err := q.Find(ctx, "user:abc123")   // by record id (accepts "abc123", "user:abc123" or models.RecordID)
user, err := q.FindOrFail(ctx, "abc123")  // returns ErrNotFound when missing
user, err := q.WhereEq("email", "x@y.com").First(ctx)
users, err := q.OrderBy("name").Limit(10).Get(ctx)
```

### Find or create

```go
// search by email, otherwise create with the extra attributes
user, err := q.FirstOrCreate(ctx,
    map[string]any{"email": "dave@example.com"},
    map[string]any{"name": "Dave"},
)

// like FirstOrCreate but returns a new (unsaved) model instead of persisting
model, err := q.FirstOrNew(ctx, map[string]any{"email": "dave@example.com"})

// find by attrs, update if found, create otherwise
user, err := q.UpdateOrCreate(ctx,
    map[string]any{"email": "dave@example.com"},
    map[string]any{"name": "Dave"},
)
```

### Updating

```go
user, err := q.Update(ctx, id, map[string]any{"name": "Dave"}) // SET-based update, bumps updated_at

// update matching records based on the current WHERE clauses
updated, err := q.WhereEq("is_active", false).UpdateWhere(ctx, map[string]any{"is_active": true})

// atomic counter updates (no read-modify-write race)
user, err := q.Increment(ctx, id, "credits", 5)
user, err = q.Decrement(ctx, id, "credits", 1)
users, err := q.WhereEq("plan", "pro").WhereIncrement(ctx, "messages_used", 1)

// insert-or-update a whole model (creates when it has no id, updates otherwise)
user, err := q.Save(ctx, user)
```

### Deleting

```go
err := q.Delete(ctx, id)        // soft delete when the model has a deleted_at field
err := q.DeleteWhere(ctx)       // deletes everything matching the current WHERE clauses
err := q.ForceDelete(ctx, id)   // permanent delete
err := q.Restore(ctx, id)       // clear deleted_at
err := q.Truncate(ctx)          // delete every record in the table
```

## Soft Deletes

When your struct has a `deleted_at` field, the ORM automatically:

- filters `deleted_at IS NONE` on every `Get`, `First`, `Find`, `Count`, `Exists` and `Paginate`;
- turns `Delete` / `DeleteWhere` into a soft delete.

```go
q := surrealgoorm.Query[User](db, "user")
q.Delete(ctx, id)               // sets deleted_at

q.Find(ctx, id)                 // nil – excluded by default
q.WithTrashed().Find(ctx, id)   // found
q.OnlyTrashed().Find(ctx, id)   // only trashed records
q.Restore(ctx, id)              // bring it back
```

## Timestamps

If the model has `created_at` / `updated_at` fields (any of `time.Time`, `*time.Time` or `string`), they are set automatically:

- `Create` sets both.
- `Update`, `Save` and `UpdateWhere` bump `updated_at`.

Use `schema.Table.Timestamps()` to define matching columns, or define them manually.

## Mass Assignment

Control which keys `CreateFromMap`, `FirstOrCreate`, `FirstOrNew` and `UpdateOrCreate` may assign using the `orm` struct tag:

```go
type User struct {
    Name     string `json:"name"`     // always assigned
    Email    string `json:"email" orm:"fillable"` // only assigned when fillable is defined
    Password string `json:"password" orm:"guarded"` // never assigned from maps
}
```

- When any field is tagged `orm:"fillable"`, only fillable fields are assigned.
- Otherwise, any field tagged `orm:"guarded"` is excluded.

## Observers (Model Events)

Implement one of these interfaces on your model to hook into the lifecycle:

| Event    | Interface        | When                              |
|----------|------------------|-----------------------------------|
| Saving   | `SavingHook`     | before create or update           |
| Creating | `CreatingHook`   | before create                     |
| Updating | `UpdatingHook`   | before update                     |
| Deleting | `DeletingHook`   | before delete                     |
| Created  | `CreatedHook`    | after create                      |
| Updated  | `UpdatedHook`    | after update                      |
| Deleted  | `DeletedHook`    | after delete                      |
| Saved    | `SavedHook`      | after create or update            |

Pre-save hooks can abort the operation by returning a non-nil error.

```go
func (u *User) Creating() error {
    if u.Email == "" {
        return errors.New("email is required")
    }
    return nil
}

func (u *User) Saved() {
    fmt.Println("user persisted:", u.ID)
}
```

## Aggregates

```go
count, err := q.WhereEq("is_active", true).Count(ctx)
exists, err := q.Exists(ctx)
total, err := q.Sum(ctx, "price")
avg,   err := q.Avg(ctx, "price")
min,   err := q.Min(ctx, "price")
max,   err := q.Max(ctx, "price")
```

Aggregates respect the current WHERE clauses and the soft-delete scope.

## Pagination

```go
page, err := q.OrderBy("name").Paginate(ctx, 2, 15)
// page.Data []T, page.Total, page.PerPage, page.CurrentPage, page.LastPage, page.From, page.To
```

## Relationships

Declare relations on the struct with `orm` tags, then hydrate them with typed `With[R]()` (Go 1.27+). Prefer that over calling `HasMany` inside a loop (N+1).

```go
type User struct {
    ID      string   `json:"id,omitempty"`
    Name    string   `json:"name"`
    Posts   []Post   `json:"posts,omitempty" orm:"has_many,table:post,fk:user_id"`
    Profile *Profile `json:"profile,omitempty" orm:"has_one,table:profile,fk:user_id"`
}

type Post struct {
    ID     string `json:"id,omitempty"`
    Title  string `json:"title"`
    UserID string `json:"user_id"`
    User   *User  `json:"user,omitempty" orm:"belongs_to,table:user,fk:user_id"`
}

type Profile struct {
    ID     string `json:"id,omitempty"`
    Bio    string `json:"bio"`
    UserID string `json:"user_id"`
}

users, err := surrealgoorm.Query[User](db, "user").With[[]Post]().Get(ctx)
for _, u := range *users {
    for _, p := range u.Posts { // typed []Post
        _ = p.Title
    }
}

post, err := surrealgoorm.Query[Post](db, "post").With[*User]().First(ctx)
_ = post.User.Name

// chain distinct relation types
users, err = surrealgoorm.Query[User](db, "user").
    With[[]Post]().
    With[*Profile]().
    Get(ctx)
```

`With[R]()` lives on `*Model[T]` (interfaces cannot declare generic methods). It picks the unique field on `T` whose type is `R`. If two fields share that type, disambiguate with `WithField`:

```go
users, err := surrealgoorm.Query[User](db, "user").
    WithField[[]Post]("Drafts").
    Get(ctx)
```

`WithNames` is the non-generic fallback (Go/json field names) and is the method on `QueryBuilder[T]`:

```go
users, err := surrealgoorm.Query[User](db, "user").
    WithNames("Posts", "Profile").
    Get(ctx)
```

| Tag | Meaning |
|-----|---------|
| `has_many,table:T,fk:F` | Child table `T`, FK column `F` on the child pointing at parent `id` |
| `has_one,table:T,fk:F` | Same as has_many, take the first related row (or empty) |
| `belongs_to,table:T,fk:F` | FK `F` on **this** record → load parent from table `T` |
| `fetch` | SurrealDB record-link field; parent `SELECT` uses `FETCH <jsonName>` |

FK-style tags require `table` and `fk`. Relation fields are omitted from `Create`/`Update` maps so nested graphs are not persisted.

Record-link example:

```go
type User struct {
    ID    string `json:"id,omitempty"`
    Name  string `json:"name"`
    Posts []Post `json:"posts,omitempty" orm:"fetch"`
}

users, err := surrealgoorm.Query[User](db, "user").With[[]Post]().Get(ctx)
```

Graph edges still use `Relate`:

```go
edge, err := q.Relate(ctx,
    models.NewRecordID("user", "alice"),
    "likes",
    models.NewRecordID("post", "p1"),
    map[string]any{"kind": "like"},
)
```

### HasMany / HasOne / BelongsTo

Foreign-key style relationships use a column on the child table that holds the
parent's id (e.g. `post.user_id`). Chain a query that is already narrowed to the
record(s) you care about into a generic method:

```go
// all posts written by Alice
posts, err := surrealgoorm.Query[User](db, "user").
    WhereEq("name", "Alice").
    HasMany[Post](ctx, "post", "user_id")

// the author of a post (matched against the parent's id field)
author, err := surrealgoorm.Query[Post](db, "post").
    WhereEq("title", "Hello").
    BelongsTo[User](ctx, "user", "user_id")
```

`HasOne` behaves like `HasMany` but returns a single record:

```go
post, err := surrealgoorm.Query[User](db, "user").
    WhereEq("name", "Bob").
    HasOne[Post](ctx, "post", "user_id")
```

Package-level `HasMany` / `HasOne` / `BelongsTo` functions remain available but
are deprecated in favor of the fluent methods above.

### Associate / Dissociate

`Associate` points a child record at a parent by setting its foreign key,
`Dissociate` clears it (sets it to `NONE`):

```go
updated, err := surrealgoorm.Query[Post](db, "post").
    WhereEq("title", "Hello").
    Associate(ctx, "user_id", bob) // *User

cleared, err := surrealgoorm.Query[Post](db, "post").
    WhereEq("title", "Hello").
    Dissociate(ctx, "user_id")
```

Note: when querying by record id, pass a `models.RecordID` (or use `Find`) — a
plain string like `"user:abc"` is not automatically cast to a record id in
SurrealDB comparisons.

## Transactions

Interactive transactions run multiple ORM operations atomically. They require a
WebSocket connection (SurrealDB v3+).

```go
tx, err := surrealgoorm.Begin(ctx, db)
if err != nil {
    return err
}
defer tx.Cancel(ctx) // cancel if not committed

user, err := tx.Query[User]("user").Create(ctx, &User{Name: "Alice"})
if err != nil {
    return err
}
if _, err := tx.Query[Post]("post").Create(ctx, &Post{Title: "Hello", UserID: user.ID}); err != nil {
    return err
}

return tx.Commit(ctx)
```

Every `tx.Query` operation runs inside the transaction; nothing is visible
outside it until `Commit` is called, and `Cancel` discards all changes.

- `Begin(ctx, db)` — start a transaction (returns `*Transaction`)
- `(*Transaction).Commit(ctx)` — make changes permanent
- `(*Transaction).Cancel(ctx)` — discard changes (safe to call multiple times)
- `(*Transaction).IsClosed()` — whether the transaction is done
- `(*Transaction).Raw(ctx, sql, vars)` — run raw SurrealQL inside the transaction
- `(*Transaction).Query[T](table)` — ORM builder scoped to the transaction
- `QueryTx[T](tx, table)` — deprecated alias for `tx.Query[T](table)`

Calling `Commit` or `Cancel` after the transaction is already closed returns
`ErrTransactionClosed`.

## Raw Queries

For DDL, aggregates or expressions the builder cannot express, use the
package-level helpers. `Raw` checks every statement's error, not just the first,
and returns the typed results.

```go
results, err := surrealgoorm.Raw[[]User](ctx, db,
    "SELECT * FROM user WHERE age > $age", map[string]any{"age": 18})

err = surrealgoorm.Exec(ctx, db, "DEFINE TABLE archived SCHEMAFULL;", nil)
```

## Query Builder

The core is a fluent, generic query builder. Chain methods return `*Model[T]`
so you can keep chaining and call Go 1.27 generic methods (e.g. `With[R]`, `HasMany[R]`).
`*Model[T]` still implements `QueryBuilder[T]`. Generic methods are not on the interface.

### Constructor

```go
surrealgoorm.Query[T](db, "table_name") // returns *Model[T]
```

### Methods

| Method | Description |
|--------|-------------|
| `Select(columns ...string)` | Select specific fields (default: all) |
| `Where(column, operator string, value any)` | Add a WHERE clause (e.g. `"age", ">", 18`) |
| `WhereEq(column string, value any)` | Shorthand for equality WHERE |
| `WhereNotNull(column string)` | WHERE field IS NOT NULL |
| `WhereNull(column string)` | WHERE field IS NULL |
| `WhereIn(column string, values []any)` | WHERE field IN (values) |
| `WhereContains(column string, value any)` | WHERE array field CONTAINS value |
| `WhereContainsAny(column string, values ...any)` | WHERE array field CONTAINSANY values |
| `OrderBy(column string, direction ...string)` | Add ORDER BY (pass `"DESC"` for descending) |
| `OrderByDesc(column string)` | Add ORDER BY field DESC |
| `Limit(limit int)` | Add LIMIT |
| `With[R]()` | Eager-load the unique relation field of type `R` (`*Model[T]` only) |
| `WithField[R](name)` | Eager-load a named relation, checking type `R` (`*Model[T]` only) |
| `WithNames(relations ...string)` | Eager-load relations by Go or json field name |
| `WithTrashed()` | Include soft-deleted records |
| `OnlyTrashed()` | Only soft-deleted records |
| `ToSQL()` | Returns the generated SurrealQL string with parameters inlined |
| `ToBuildSQL()` | Returns SQL and vars for `surrealdb.Query` |
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

// Debug generated SQL (parameters inlined)
sql := surrealgoorm.Query[User](db, "user").
    WhereEq("email", "test@test.com").
    ToSQL()
fmt.Println(sql) // SELECT * FROM user WHERE type::field('email') = 'test@test.com'
```

## Schema Builder

Define SurrealDB tables and columns using a DSL.

### Creating a Table

```go
schema := surrealgoorm.Schema{Db: db}

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
    defer db.Close(context.Background())

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

### Interface

```go
QueryBuilder[T any] interface {
    Select(columns ...string) *Model[T]
    Where(column, operator string, value any) *Model[T]
    WhereEq(column string, value any) *Model[T]
    WhereNotNull(column string) *Model[T]
    WhereNull(column string) *Model[T]
    WhereIn(column string, values []any) *Model[T]
    WhereContains(column string, value any) *Model[T]
    WhereContainsAny(column string, values ...any) *Model[T]
    OrderBy(column string, direction ...string) *Model[T]
    OrderByDesc(column string) *Model[T]
    Limit(limit int) *Model[T]
    WithNames(relations ...string) *Model[T]
    WithTrashed() *Model[T]
    OnlyTrashed() *Model[T]
    ToSQL() string
    ToBuildSQL() (string, map[string]any)
    First(ctx context.Context) (*T, error)
    Get(ctx context.Context) (*[]T, error)

    Create(ctx context.Context, data *T) (*T, error)
    CreateWithID(ctx context.Context, id any, data *T) (*T, error)
    CreateMany(ctx context.Context, data []T) ([]T, error)
    CreateFromMap(ctx context.Context, attrs map[string]any) (*T, error)
    Find(ctx context.Context, id any) (*T, error)
    FindOrFail(ctx context.Context, id any) (*T, error)
    FirstOrCreate(ctx context.Context, attrs map[string]any, values ...map[string]any) (*T, error)
    FirstOrNew(ctx context.Context, attrs map[string]any, values ...map[string]any) (*T, error)
    UpdateOrCreate(ctx context.Context, attrs map[string]any, values map[string]any) (*T, error)
    Save(ctx context.Context, model *T) (*T, error)
    Update(ctx context.Context, id any, data map[string]any) (*T, error)
    UpdateWhere(ctx context.Context, data map[string]any) ([]T, error)
    Increment(ctx context.Context, id any, column string, by int64) (*T, error)
    Decrement(ctx context.Context, id any, column string, by int64) (*T, error)
    WhereIncrement(ctx context.Context, column string, by int64) ([]T, error)
    WhereDecrement(ctx context.Context, column string, by int64) ([]T, error)
    Delete(ctx context.Context, id any) error
    DeleteWhere(ctx context.Context) error
    ForceDelete(ctx context.Context, id any) error
    Restore(ctx context.Context, id any) error
    Truncate(ctx context.Context) error

    Count(ctx context.Context) (int, error)
    Exists(ctx context.Context) (bool, error)
    Sum(ctx context.Context, column string) (float64, error)
    Avg(ctx context.Context, column string) (float64, error)
    Min(ctx context.Context, column string) (float64, error)
    Max(ctx context.Context, column string) (float64, error)
    Paginate(ctx context.Context, page, perPage int) (*Pagination[T], error)

    Relate(ctx context.Context, from models.RecordID, edge string, to models.RecordID, content map[string]any) (map[string]any, error)
}
```

### Types

```go
Model[T any]         // concrete implementation of QueryBuilder[T]; hosts generic relationship methods
Pagination[T any]    // page results (Data, Total, PerPage, CurrentPage, LastPage, From, To)
Schema               // table definition builder (holds *surrealdb.DB)
Table                // table definition (Name string, Columns []*Column)
Column               // column definition (Name, Type, IsUnique, DefaultValue, Optional)
Transaction          // interactive transaction (Begin/Commit/Cancel/IsClosed/Raw/Query)
ErrNotFound          // returned by FindOrFail when no record matches
ErrTransactionClosed // returned by Commit/Cancel on a closed transaction
```

### Constructors

```go
func Query[T any](db *surrealdb.DB, table string) *Model[T]
func Raw[T any](ctx context.Context, db *surrealdb.DB, sql string, vars map[string]any) (*[]surrealdb.QueryResult[T], error)
func Exec(ctx context.Context, db *surrealdb.DB, sql string, vars map[string]any) error
func Begin(ctx context.Context, db *surrealdb.DB) (*Transaction, error)
func (t *Transaction) Query[T any](table string) *Model[T]
func QueryTx[T any](tx *Transaction, table string) *Model[T] // Deprecated: use tx.Query[T]
```

### Relationship methods (on `*Model[T]`)

```go
func (m *Model[T]) With[R any]() *Model[T]
func (m *Model[T]) WithField[R any](name string) *Model[T]
func (m Model[T]) WithNames(names ...string) *Model[T]
func (m *Model[T]) HasMany[R any](ctx context.Context, relatedTable, foreignKey string) (*[]R, error)
func (m *Model[T]) HasOne[R any](ctx context.Context, relatedTable, foreignKey string) (*R, error)
func (m *Model[T]) BelongsTo[R any](ctx context.Context, parentTable, foreignKey string) (*R, error)
func (m *Model[T]) Associate[R any](ctx context.Context, foreignKey string, parent *R) (*T, error)
func (m *Model[T]) Dissociate(ctx context.Context, foreignKey string) (*T, error)
```

### Deprecated package-level relationship helpers

```go
func HasMany[T, R any](ctx context.Context, parent QueryBuilder[T], relatedTable, foreignKey string) (*[]R, error)
func HasOne[T, R any](ctx context.Context, parent QueryBuilder[T], relatedTable, foreignKey string) (*R, error)
func BelongsTo[T, R any](ctx context.Context, child QueryBuilder[T], parentTable, foreignKey string) (*R, error)
func Associate[T, R any](ctx context.Context, child QueryBuilder[T], foreignKey string, parent *R) (*T, error)
func Dissociate[T any](ctx context.Context, child QueryBuilder[T], foreignKey string) (*T, error)
```

## Integration Tests

The package includes integration tests against a live SurrealDB. Set these environment variables and run with:

```bash
SURREALDB_INTEGRATION=1 \
SURREALDB_URL=ws://localhost:8000/rpc \
SURREALDB_USER=root \
SURREALDB_PASS=root \
SURREALDB_NS=test \
SURREALDB_DB=test \
go test -run Integration ./...
```
