# Models

Define your database tables as Go structs.

## Base Model

Embed `Model` to get built-in fields:

```go
type User struct {
    surrealgoorm.Model
    Name  string `orm:"column:name"`
    Email string `orm:"column:email"`
}
```

This gives you:
- `ID` - SurrealDB record ID
- `CreatedAt` - auto-set on create
- `UpdatedAt` - auto-set on update

## Soft Delete

Add soft delete support with `SoftDeletesModel`:

```go
type Post struct {
    surrealgoorm.Model
    Title string `orm:"column:title"`
    surrealgoorm.SoftDeletesModel
}
```

This adds `DeletedAt` field. Records with a non-nil `DeletedAt` are considered deleted but kept in the database.

The ORM's `QueryModel` automatically filters deleted records. Use `.WithTrashed()` to include them.

## Table Name

The ORM generates table names automatically from struct names:

| Struct | Table |
|--------|-------|
| User | users |
| Post | posts |
| APIKey | api_keys |
| Category | categories |

Override with `TableName()`:

```go
func (User) TableName() string {
    return "app_users"
}
```

## Field Tags

Use `orm:` tags to customize field mapping:

```go
type User struct {
    ID        models.RecordID `orm:"column:id;primary"`
    Name      string         `orm:"column:name"`
    Email     string         `orm:"column:email;unique"`
    CreatedAt time.Time      `orm:"column:created_at;auto"`
    UpdatedAt time.Time      `orm:"column:updated_at;auto"`
}
```

### Tag Options

- `column:name` - database column name
- `primary` - mark as primary key
- `auto` - auto-generate on create/update (for timestamps)
- `unique` - create unique index

## Relations

Define relationships between models.

### Belongs To

```go
type Post struct {
    surrealgoorm.Model
    Title    string `orm:"column:title"`
    AuthorID string `orm:"column:author_id"`
    Author   *User  `orm:"belongs_to:users;foreign_key:AuthorID"`
}
```

### Has Many

```go
type User struct {
    surrealgoorm.Model
    Name  string `orm:"column:name"`
    Posts []Post `orm:"has_many:posts;foreign_key:author_id"`
}
```

### Many to Many

Requires a pivot table:

```go
type User struct {
    surrealgoorm.Model
    Name   string    `orm:"column:name"`
    Roles  []Role    `orm:"many_to_many:roles;through:user_roles;source_foreign_key:user_id;target_foreign_key:role_id"`
}

type Role struct {
    surrealgoorm.Model
    Name string `orm:"column:name"`
}

type UserRole struct {
    surrealgoorm.Model
    UserID string `orm:"column:user_id"`
    RoleID string `orm:"column:role_id"`
}
```

## Full Example

```go
import (
    "time"

    "github.com/kemboi22/surreal-go-orm"
    "github.com/surrealdb/surrealdb.go/pkg/models"
)

type User struct {
    surrealgoorm.Model
    Name     string    `orm:"column:name"`
    Email    string   `orm:"column:email;unique"`
    Age      int      `orm:"column:age"`
    Posts    []Post   `orm:"has_many:posts;foreign_key:author_id"`
}

func (User) TableName() string { return "users" }

type Post struct {
    surrealgoorm.Model
    Title     string   `orm:"column:title"`
    Content   string   `orm:"column:content"`
    AuthorID  string   `orm:"column:author_id"`
    Author    *User    `orm:"belongs_to:users;foreign_key:AuthorID"`
    Tags      []Tag    `orm:"many_to_many:tags;through:post_tags;source_foreign_key:post_id;target_foreign_key:tag_id"`
    surrealgoorm.SoftDeletesModel
}

func (Post) TableName() string { return "posts" }

type Tag struct {
    surrealgoorm.Model
    Name string `orm:"column:name"`
}

func (Tag) TableName() string { return "tags" }

type PostTag struct {
    surrealgoorm.Model
    PostID string `orm:"column:post_id"`
    TagID  string `orm:"column:tag_id"`
}
```

## Model Interfaces

Your models can implement these interfaces:

### TableNameGetter

```go
type TableNameGetter interface {
    TableName() string
}
```

### PrimaryKeyGetter

```go
type PrimaryKeyGetter interface {
    GetID() models.RecordID
}
```

## Timestamps

The ORM auto-sets `CreatedAt` and `UpdatedAt` when you embed `Model`:

- `CreatedAt` - set only on create
- `UpdatedAt` - set on create and update

Customize the column names with tags:

```go
type User struct {
    surrealgoorm.Model
    CreatedAt  time.Time `orm:"column:created_at;auto"`
    UpdatedAt  time.Time `orm:"column:updated_at;auto"`
}
```