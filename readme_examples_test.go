package surrealgoorm_test

import (
	"context"
	"testing"
	"time"

	surrealgoorm "github.com/kemboi22/surreal-go-orm"
	"github.com/kemboi22/surreal-go-orm/migrator"
	"github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

type User struct {
	ID        string     `json:"id,omitempty"`
	Name      string     `json:"name"`
	Email     string     `json:"email"`
	Age       int        `json:"age"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at"`
	Posts     []Post     `json:"posts,omitempty" orm:"has_many,table:post,fk:user_id"`
	Profile   *Profile   `json:"profile,omitempty" orm:"has_one,table:profile,fk:user_id"`
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

func TestReadmeUsageCompiles(t *testing.T) {
	t.Skip("compile-only check; mirrors README examples")
	db := &surrealdb.DB{}
	ctx := context.Background()

	q := surrealgoorm.Query[User](db, "user")
	_ = q

	// Query builder
	users, err := surrealgoorm.Query[User](db, "user").
		Where("age", ">", 18).
		WhereEq("is_active", true).
		OrderBy("name").
		Limit(10).
		Get(ctx)
	_, _ = users, err

	sql := surrealgoorm.Query[User](db, "user").
		WhereEq("email", "test@test.com").
		ToSQL()
	_ = sql

	_, _ = surrealgoorm.Query[User](db, "user").WithTrashed().Get(ctx)
	_, _ = surrealgoorm.Query[User](db, "user").OnlyTrashed().Get(ctx)

	// Eloquent CRUD
	u := &User{Name: "Alice", Email: "alice@example.com"}
	created, err := surrealgoorm.Query[User](db, "user").Create(ctx, u)
	_, _ = created, err

	_, err = surrealgoorm.Query[User](db, "user").CreateMany(ctx, []User{*u})
	_, err = surrealgoorm.Query[User](db, "user").CreateFromMap(ctx, map[string]any{"name": "Bob", "email": "bob@example.com"})
	_, err = surrealgoorm.Query[User](db, "user").CreateWithID(ctx, "bob", &User{Name: "Bob"})

	_, err = q.Find(ctx, "user:abc123")
	_, err = q.FindOrFail(ctx, "abc123")
	_, err = q.WhereEq("email", "x@y.com").First(ctx)

	_, err = q.FirstOrCreate(ctx, map[string]any{"email": "dave@example.com"}, map[string]any{"name": "Dave"})
	_, err = q.FirstOrNew(ctx, map[string]any{"email": "dave@example.com"})
	_, err = q.UpdateOrCreate(ctx, map[string]any{"email": "dave@example.com"}, map[string]any{"name": "Dave"})

	_, err = q.Update(ctx, created.ID, map[string]any{"name": "Dave"})
	_, err = q.WhereEq("is_active", false).UpdateWhere(ctx, map[string]any{"is_active": true})
	_, err = q.Save(ctx, u)

	err = q.Delete(ctx, created.ID)
	err = q.DeleteWhere(ctx)
	err = q.ForceDelete(ctx, created.ID)
	err = q.Restore(ctx, created.ID)
	err = q.Truncate(ctx)

	// Aggregates
	_, err = q.WhereEq("is_active", true).Count(ctx)
	_, err = q.Exists(ctx)
	_, err = q.Sum(ctx, "price")
	_, err = q.Avg(ctx, "price")
	_, err = q.Min(ctx, "price")
	_, err = q.Max(ctx, "price")

	// Pagination
	page, err := q.OrderBy("name").Paginate(ctx, 2, 15)
	_, _ = page.Data, page.Total
	_, _ = page.PerPage, page.CurrentPage
	_, _ = page.LastPage, page.From
	_, _ = page.To, err

	// Relationships
	_, err = surrealgoorm.Query[User](db, "user").
		WhereEq("email", "bob@test.com").
		With[[]Post]().
		First(ctx)
	_, err = surrealgoorm.Query[Post](db, "post").With[*User]().First(ctx)
	_, err = surrealgoorm.Query[User](db, "user").With[[]Post]().With[*Profile]().Get(ctx)
	_, err = surrealgoorm.Query[User](db, "user").WithField[[]Post]("Posts").Get(ctx)
	_, err = surrealgoorm.Query[User](db, "user").WithNames("Posts", "Profile").Get(ctx)

	_, err = q.Relate(ctx,
		models.NewRecordID("user", "alice"),
		"likes",
		models.NewRecordID("post", "p1"),
		map[string]any{"kind": "like"},
	)

	_, err = surrealgoorm.Query[Post](db, "post").
		WhereEq("user_id", models.NewRecordID("user", "alice")).
		Get(ctx)
	_ = err
}

func TestReadmeMigrationsCompile(t *testing.T) {
	t.Skip("compile-only check; mirrors README examples")
	am := migrator.NewAutoMigrator(&surrealdb.DB{})
	_ = am
	err := am.AutoMigrate(context.Background(), []surrealgoorm.Migration{
		CreateUserTable{},
	})
	_ = err
}

func TestReadmeSchemaCompiles(t *testing.T) {
	t.Skip("compile-only check; mirrors README examples")
	db := &surrealdb.DB{}
	ctx := context.Background()

	schema := surrealgoorm.Schema{Db: db}
	err := schema.CreateTable(ctx, "user", func(t *surrealgoorm.Table) {
		t.ID()
		t.String("name")
		t.String("email").Unique()
		t.Int("age").Default(18).Nullable()
		t.Bool("is_active").Default(true)
		t.Float("score")
		t.Timestamps()
	})
	_ = err

	table := surrealgoorm.Table{Name: "user"}
	table.ID()
	table.String("name")
	built := table.Build()
	_ = built
}
