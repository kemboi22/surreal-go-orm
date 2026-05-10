package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/surrealdb/surrealdb.go/pkg/models"

	surrealgoorm "github.com/kemboi22/surreal-go-orm"
	ormtypes "github.com/kemboi22/surreal-go-orm/types"
)

type User struct {
	surrealgoorm.Model
	Name  string `orm:"column:name"`
	Email string `orm:"column:email"`
	Age   int    `orm:"column:age"`
	Posts []Post `orm:"has_many:posts;foreign_key:author_id"`
	surrealgoorm.Timestamps
}

func (User) TableName() string {
	return "users"
}

type Post struct {
	surrealgoorm.Model
	Title    string `orm:"column:title"`
	Content  string `orm:"column:content"`
	AuthorID string `orm:"column:author_id"`
	Author   *User  `orm:"belongs_to:users;foreign_key:AuthorID"`
	surrealgoorm.SoftDeletesModel
}

func (Post) TableName() string {
	return "posts"
}

func main() {
	fmt.Println("Example: Surreal Go ORM")
	fmt.Println("Run example/migrations.sql first if you want a quick setup.")

	ctx := context.Background()

	dbName := "main"
	nsName := "main"
	db, err := surrealgoorm.Connect(ctx, surrealgoorm.Config{
		URL:       "ws://localhost:8000",
		Username:  "root",
		Password:  "root",
		Namespace: &nsName,
		Database:  &dbName,
	})
	if err != nil {
		log.Fatalf("connect failed: %v", err)
	}
	defer func() {
		if err := db.Close(ctx); err != nil {
			log.Fatalf("close failed: %v", err)
		}
	}()

	showSchemaExample()
	err = runMigrations(ctx, db)
	if err != nil {
		fmt.Printf("Err on migrations: %v", err)
	}
	runCRUD(ctx, db)
	runQueryBuilder(ctx, db)
	runRelations(ctx, db)
	runTransactions(ctx, db)
}

func showSchemaExample() {
	schema := surrealgoorm.NewTableSchema("users").
		AddField(ormtypes.String("name").Index().Build()).
		AddField(ormtypes.String("email").Unique().Build())

	statements, err := schema.Statements()
	if err != nil {
		log.Printf("schema build failed: %v", err)
		return
	}
	fmt.Println("\n=== Schema Builder ===")
	for _, statement := range statements {
		fmt.Println(statement)
	}
}

func runMigrations(ctx context.Context, db *surrealgoorm.DB) error {
	fmt.Println("\n=== AutoMigrate ===")

	migrator := surrealgoorm.NewMigrator(db)

	schemas := []*surrealgoorm.TableSchema{
		surrealgoorm.NewTableSchema("users").
			SchemaFull().
			AddField(ormtypes.Timestamp("created_at").Default("time::now()").Build()).
			AddField(ormtypes.Timestamp("updated_at").Default("time::now()").Build()).
			AddField(ormtypes.String("name").Build()).
			AddField(ormtypes.String("email").Unique().Build()).
			AddField(ormtypes.Integer("age").Build()).
			AddField(ormtypes.Boolean("active").Default(true).Build()),

		surrealgoorm.NewTableSchema("posts").
			SchemaFull().
			AddField(ormtypes.Timestamp("created_at").Default("time::now()").Build()).
			AddField(ormtypes.Timestamp("updated_at").Default("time::now()").Build()).
			AddField(ormtypes.String("title").Build()).
			AddField(ormtypes.String("content").Build()).
			AddField(ormtypes.Record("author_id", "users").Build()).
			AddField(ormtypes.Boolean("published").Default(false).Build()).
			AddField(ormtypes.Timestamp("deleted_at").Nullable().Build()),
	}
	for sc := range schemas {
		st, _ := schemas[sc].Statements()
		fmt.Printf("%v \n", st)
	}
	if err := migrator.AutoMigrate(ctx, schemas...); err != nil {
		return fmt.Errorf("auto migrate failed: %w", err)
	}

	fmt.Println("auto migrate completed successfully")
	return nil
}
func runCRUD(ctx context.Context, db *surrealgoorm.DB) {
	fmt.Println("\n=== CRUD ===")

	user := &User{
		Model: surrealgoorm.Model{ID: models.NewRecordID("users", "john")},
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   30,
	}

	user1 := &User{
		Model: surrealgoorm.Model{ID: models.NewRecordID("users", "kemboi")},
		Name:  "Kemboi ELvis",
		Email: "kemboielvis22@gmail.com",
		Age:   22,
	}
	if err := surrealgoorm.Create(ctx, db, user); err != nil {
		log.Printf("create failed: %v", err)
	}
	if err := surrealgoorm.Create(ctx, db, user1); err != nil {
		log.Printf("create failed: %v", err)
	}

	var found User
	if err := surrealgoorm.Find(ctx, db, "users:john", &found); err != nil {
		log.Printf("find failed: %v", err)
	} else {
		fmt.Printf("found user: %+v\n", found)
	}

	found.Age = 31
	if err := surrealgoorm.Update(ctx, db, &found); err != nil {
		log.Printf("update failed: %v", err)
	}
}

func runQueryBuilder(ctx context.Context, db *surrealgoorm.DB) {
	fmt.Println("\n=== Query Builder ===")

	if err := db.Query("posts").AllowAll().ForceDelete(ctx); err != nil {
		log.Printf("cleanup failed: %v", err)
	}

	posts := []map[string]any{
		{
			"title":   "Hello World",
			"content": "First post",
			// "author_id":  "users:john",
			"deleted_at": time.Now(),
		},
		{
			"title":   "Go ORM",
			"content": "Second post",
			// "author_id":  "users:john",
			"deleted_at": time.Now(),
		},
	}
	if err := db.Query("posts").InsertMany(ctx, posts); err != nil {
		log.Printf("insert many failed: %v", err)
	}

	var filtered []Post
	err := db.Query("posts").
		SoftDeletes().
		WhereLike("title", "Go").
		OrderByDesc("created_at").
		All(ctx, &filtered)
	if err != nil {
		log.Printf("query failed: %v", err)
	} else {
		fmt.Printf("filtered posts: %+v\n", filtered)
	}

	if err := db.Query("posts").
		SoftDeletes().
		Where("id", "=", "posts:hello-world").
		Delete(ctx); err != nil {
		log.Printf("soft delete failed: %v", err)
	}
}

func runRelations(ctx context.Context, db *surrealgoorm.DB) {
	fmt.Println("\n=== Relations ===")

	var users []User
	err := surrealgoorm.QueryModel[User](ctx, db).
		With("posts.author").
		All(ctx, &users)
	if err != nil {
		log.Printf("relation load failed: %v", err)
		return
	}

	fmt.Printf("users with posts.author: %+v\n", users)
}

func runTransactions(ctx context.Context, db *surrealgoorm.DB) {
	fmt.Println("\n=== Transactions ===")

	err := db.Transaction(ctx, func(tx *surrealgoorm.DB) error {
		return tx.Query("users").
			AllowAll().
			Update(ctx, map[string]any{"age": 32})
	})
	if err != nil {
		log.Printf("transaction failed: %v", err)
	}
}
