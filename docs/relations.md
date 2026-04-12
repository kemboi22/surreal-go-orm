# Relations

## Tags

Relations use the `orm` tag.

Examples:

```go
Posts  []Post `orm:"has_many:posts;foreign_key:author_id"`
Profile *Profile `orm:"has_one:profiles;foreign_key:user_id"`
Author *User `orm:"belongs_to:users;foreign_key:AuthorID"`
```

## Has Many

```go
type User struct {
	surrealgoorm.Model
	Name  string `orm:"column:name"`
	Posts []Post `orm:"has_many:posts;foreign_key:author_id"`
}

type Post struct {
	surrealgoorm.Model
	Title    string `orm:"column:title"`
	AuthorID string `orm:"column:author_id"`
}
```

## Belongs To

```go
type Post struct {
	surrealgoorm.Model
	Title    string `orm:"column:title"`
	AuthorID string `orm:"column:author_id"`
	Author   *User  `orm:"belongs_to:users;foreign_key:AuthorID"`
}
```

## Load Relations

With query:

```go
var users []User
err := surrealgoorm.QueryModel[User](ctx, db).
	With("posts").
	All(ctx, &users)
```

On loaded models:

```go
var users []User
err := surrealgoorm.All(ctx, db, &users)
if err != nil {
	return err
}

err = db.With(ctx, &users, "posts")
```
