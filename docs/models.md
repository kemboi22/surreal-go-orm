# Models

## Base Model

```go
type Model struct {
	ID        models.RecordID `orm:"column:id;primary"`
	CreatedAt time.Time       `orm:"column:created_at;auto"`
	UpdatedAt time.Time       `orm:"column:updated_at;auto"`
}
```

## Soft Delete

Soft delete is opt-in.

Add `SoftDeletes` or `SoftDeletesModel` only on models that need it.

```go
type Post struct {
	surrealgoorm.Model
	Title string `orm:"column:title"`
	surrealgoorm.SoftDeletesModel
}
```

## Table Name

Set it when you want full control.

```go
func (User) TableName() string {
	return "users"
}
```

If you skip it, the ORM builds a snake_case plural name.

Examples:
- `User` -> `users`
- `APIKey` -> `api_keys`
- `Category` -> `categories`

## Tags

- ``orm:"column:name"`` maps a field to a column
- ``orm:"column:id;primary"`` marks a primary key field
- ``orm:"column:created_at;auto"`` skips write on auto fields
- ``orm:"has_many:posts;foreign_key:author_id"`` defines a relation

## Example

```go
type User struct {
	surrealgoorm.Model
	Name  string `orm:"column:name"`
	Email string `orm:"column:email"`
}

type Post struct {
	surrealgoorm.Model
	Title    string `orm:"column:title"`
	AuthorID string `orm:"column:author_id"`
	surrealgoorm.SoftDeletesModel
}
```
