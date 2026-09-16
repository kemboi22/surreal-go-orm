package surrealgoorm

import (
	"context"
	"testing"

	"github.com/surrealdb/surrealdb.go"
)

type relUser struct {
	ID    string    `json:"id,omitempty"`
	Name  string    `json:"name"`
	Posts []relPost `json:"posts,omitempty" orm:"has_many,table:test_rel_posts,fk:user_id"`
}

type relPost struct {
	ID      string   `json:"id,omitempty"`
	Title   string   `json:"title"`
	UserID  string   `json:"user_id"`
	Comment string   `json:"comment"`
	User    *relUser `json:"user,omitempty" orm:"belongs_to,table:test_rel_users,fk:user_id"`
}

func TestIntegrationRelationsHasManyBelongsTo(t *testing.T) {
	skipIfNoDB(t)
	db := getTestDB(t)
	defer db.Close(context.Background())
	ctx := context.Background()

	userTable := "test_rel_users"
	postTable := "test_rel_posts"
	defineEloquentSchema(t, ctx, db, userTable, []string{"name"})
	defineEloquentSchema(t, ctx, db, postTable, []string{"title", "user_id", "comment"})
	defer dropTable(t, ctx, db, userTable)
	defer dropTable(t, ctx, db, postTable)

	alice, err := Query[relUser](db, userTable).Create(ctx, &relUser{Name: "Alice"})
	if err != nil {
		t.Fatalf("Create user failed: %v", err)
	}
	bob, err := Query[relUser](db, userTable).Create(ctx, &relUser{Name: "Bob"})
	if err != nil {
		t.Fatalf("Create user failed: %v", err)
	}

	if _, err := Query[relPost](db, postTable).Create(ctx, &relPost{Title: "Hello", UserID: alice.ID}); err != nil {
		t.Fatalf("Create post failed: %v", err)
	}
	if _, err := Query[relPost](db, postTable).Create(ctx, &relPost{Title: "World", UserID: alice.ID}); err != nil {
		t.Fatalf("Create post failed: %v", err)
	}
	if _, err := Query[relPost](db, postTable).Create(ctx, &relPost{Title: "Mine", UserID: bob.ID}); err != nil {
		t.Fatalf("Create post failed: %v", err)
	}

	posts, err := Query[relUser](db, userTable).
		WhereEq("name", "Alice").
		HasMany[relPost](ctx, postTable, "user_id")
	if err != nil {
		t.Fatalf("HasMany failed: %v", err)
	}
	if posts == nil || len(*posts) != 2 {
		t.Fatalf("expected 2 posts for Alice, got %d", len(*posts))
	}
	if (*posts)[0].UserID != alice.ID && (*posts)[1].UserID != alice.ID {
		t.Error("HasMany returned posts not owned by Alice")
	}

	one, err := Query[relUser](db, userTable).
		WhereEq("name", "Bob").
		HasOne[relPost](ctx, postTable, "user_id")
	if err != nil {
		t.Fatalf("HasOne failed: %v", err)
	}
	if one == nil || one.Title != "Mine" {
		t.Fatalf("expected Bob's post 'Mine', got %+v", one)
	}

	author, err := Query[relPost](db, postTable).
		WhereEq("title", "Hello").
		BelongsTo[relUser](ctx, userTable, "user_id")
	if err != nil {
		t.Fatalf("BelongsTo failed: %v", err)
	}
	if author == nil || author.Name != "Alice" {
		t.Fatalf("expected author Alice, got %+v", author)
	}

	dissociated, err := Query[relPost](db, postTable).
		WhereEq("title", "Hello").
		Dissociate(ctx, "user_id")
	if err != nil {
		t.Fatalf("Dissociate failed: %v", err)
	}
	if dissociated.UserID != "" {
		t.Errorf("expected cleared user_id, got %q", dissociated.UserID)
	}

	reassociated, err := Query[relPost](db, postTable).
		WhereEq("title", "Hello").
		Associate(ctx, "user_id", bob)
	if err != nil {
		t.Fatalf("Associate failed: %v", err)
	}
	if reassociated.UserID != bob.ID {
		t.Errorf("expected user_id %s, got %q", bob.ID, reassociated.UserID)
	}

	bobPosts, err := Query[relUser](db, userTable).
		WhereEq("name", "Bob").
		HasMany[relPost](ctx, postTable, "user_id")
	if err != nil {
		t.Fatalf("HasMany failed: %v", err)
	}
	if bobPosts == nil || len(*bobPosts) != 2 {
		t.Fatalf("expected Bob to own 2 posts after reassociation, got %d", len(*bobPosts))
	}
}

func TestIntegrationEagerWithTypedPosts(t *testing.T) {
	skipIfNoDB(t)
	db := getTestDB(t)
	defer db.Close(context.Background())
	ctx := context.Background()

	userTable := "test_rel_users"
	postTable := "test_rel_posts"
	defineEloquentSchema(t, ctx, db, userTable, []string{"name"})
	defineEloquentSchema(t, ctx, db, postTable, []string{"title", "user_id", "comment"})
	defer dropTable(t, ctx, db, userTable)
	defer dropTable(t, ctx, db, postTable)

	alice, err := Query[relUser](db, userTable).Create(ctx, &relUser{Name: "Alice"})
	if err != nil {
		t.Fatalf("Create user failed: %v", err)
	}
	if _, err := Query[relPost](db, postTable).Create(ctx, &relPost{Title: "Hello", UserID: alice.ID}); err != nil {
		t.Fatalf("Create post failed: %v", err)
	}
	if _, err := Query[relPost](db, postTable).Create(ctx, &relPost{Title: "World", UserID: alice.ID}); err != nil {
		t.Fatalf("Create post failed: %v", err)
	}

	users, err := Query[relUser](db, userTable).
		WhereEq("name", "Alice").
		With[[]relPost]().
		Get(ctx)
	if err != nil {
		t.Fatalf("With[[]relPost] Get failed: %v", err)
	}
	if users == nil || len(*users) != 1 {
		t.Fatalf("expected 1 user, got %v", users)
	}
	u := (*users)[0]
	if len(u.Posts) != 2 {
		t.Fatalf("expected 2 typed posts on user, got %d", len(u.Posts))
	}
	for _, p := range u.Posts {
		if p.Title == "" {
			t.Error("expected post title to be populated")
		}
		if p.UserID != alice.ID {
			t.Errorf("expected post.UserID %s, got %s", alice.ID, p.UserID)
		}
	}

	post, err := Query[relPost](db, postTable).
		WhereEq("title", "Hello").
		With[*relUser]().
		First(ctx)
	if err != nil {
		t.Fatalf("With[*relUser] First failed: %v", err)
	}
	if post == nil || post.User == nil || post.User.Name != "Alice" {
		t.Fatalf("expected nested author Alice, got %+v", post)
	}
}

type relProfile struct {
	ID     string `json:"id,omitempty"`
	Bio    string `json:"bio"`
	UserID string `json:"user_id"`
}

type relUserWithProfile struct {
	ID      string      `json:"id,omitempty"`
	Name    string      `json:"name"`
	Posts   []relPost   `json:"posts,omitempty" orm:"has_many,table:test_rel_chain_posts,fk:user_id"`
	Profile *relProfile `json:"profile,omitempty" orm:"has_one,table:test_rel_chain_profiles,fk:user_id"`
}

func TestIntegrationEagerHasOneAndChain(t *testing.T) {
	skipIfNoDB(t)
	db := getTestDB(t)
	defer db.Close(context.Background())
	ctx := context.Background()

	userTable := "test_rel_chain_users"
	postTable := "test_rel_chain_posts"
	profileTable := "test_rel_chain_profiles"
	defineEloquentSchema(t, ctx, db, userTable, []string{"name"})
	defineEloquentSchema(t, ctx, db, postTable, []string{"title", "user_id", "comment"})
	defineEloquentSchema(t, ctx, db, profileTable, []string{"bio", "user_id"})
	defer dropTable(t, ctx, db, userTable)
	defer dropTable(t, ctx, db, postTable)
	defer dropTable(t, ctx, db, profileTable)

	alice, err := Query[relUserWithProfile](db, userTable).Create(ctx, &relUserWithProfile{Name: "Alice"})
	if err != nil {
		t.Fatalf("Create user failed: %v", err)
	}
	if _, err := Query[relPost](db, postTable).Create(ctx, &relPost{Title: "Hello", UserID: alice.ID}); err != nil {
		t.Fatalf("Create post failed: %v", err)
	}
	if _, err := Query[relProfile](db, profileTable).Create(ctx, &relProfile{Bio: "hi", UserID: alice.ID}); err != nil {
		t.Fatalf("Create profile failed: %v", err)
	}

	users, err := Query[relUserWithProfile](db, userTable).
		WhereEq("name", "Alice").
		With[[]relPost]().
		With[*relProfile]().
		Get(ctx)
	if err != nil {
		t.Fatalf("chained With Get failed: %v", err)
	}
	if users == nil || len(*users) != 1 {
		t.Fatalf("expected 1 user, got %v", users)
	}
	u := (*users)[0]
	if len(u.Posts) != 1 || u.Posts[0].Title != "Hello" {
		t.Fatalf("expected typed posts, got %+v", u.Posts)
	}
	if u.Profile == nil || u.Profile.Bio != "hi" {
		t.Fatalf("expected typed profile, got %+v", u.Profile)
	}
}

type fetchRelPost struct {
	ID    string `json:"id,omitempty"`
	Title string `json:"title"`
}

type fetchRelUser struct {
	ID    string         `json:"id,omitempty"`
	Name  string         `json:"name"`
	Posts []fetchRelPost `json:"posts,omitempty" orm:"fetch"`
}

func TestIntegrationEagerFetch(t *testing.T) {
	skipIfNoDB(t)
	db := getTestDB(t)
	defer db.Close(context.Background())
	ctx := context.Background()

	userTable := "test_rel_fetch_users"
	postTable := "test_rel_fetch_posts"
	defineEloquentSchema(t, ctx, db, userTable, []string{"name", "posts"})
	defineEloquentSchema(t, ctx, db, postTable, []string{"title"})
	defer dropTable(t, ctx, db, userTable)
	defer dropTable(t, ctx, db, postTable)

	post, err := Query[fetchRelPost](db, postTable).Create(ctx, &fetchRelPost{Title: "Linked"})
	if err != nil {
		t.Fatalf("Create post failed: %v", err)
	}

	sql := "CREATE " + userTable + " SET name = 'Alice', posts = [" + post.ID + "];"
	if _, err := surrealdb.Query[any](ctx, db, sql, nil); err != nil {
		t.Fatalf("Create user with record links failed: %v", err)
	}

	users, err := Query[fetchRelUser](db, userTable).
		With[[]fetchRelPost]().
		Get(ctx)
	if err != nil {
		t.Fatalf("With fetch Get failed: %v", err)
	}
	if users == nil || len(*users) != 1 {
		t.Fatalf("expected 1 user, got %v", users)
	}
	u := (*users)[0]
	if len(u.Posts) != 1 || u.Posts[0].Title != "Linked" {
		t.Fatalf("expected FETCH to hydrate posts, got %+v", u.Posts)
	}
}

func TestIntegrationRelationsDeprecatedWrappers(t *testing.T) {
	skipIfNoDB(t)
	db := getTestDB(t)
	defer db.Close(context.Background())
	ctx := context.Background()

	userTable := "test_rel_depr_users"
	postTable := "test_rel_depr_posts"
	defineEloquentSchema(t, ctx, db, userTable, []string{"name"})
	defineEloquentSchema(t, ctx, db, postTable, []string{"title", "user_id", "comment"})
	defer dropTable(t, ctx, db, userTable)
	defer dropTable(t, ctx, db, postTable)

	alice, err := Query[relUser](db, userTable).Create(ctx, &relUser{Name: "Alice"})
	if err != nil {
		t.Fatalf("Create user failed: %v", err)
	}
	if _, err := Query[relPost](db, postTable).Create(ctx, &relPost{Title: "Hello", UserID: alice.ID}); err != nil {
		t.Fatalf("Create post failed: %v", err)
	}

	posts, err := HasMany[relUser, relPost](ctx,
		Query[relUser](db, userTable).WhereEq("name", "Alice"),
		postTable, "user_id")
	if err != nil {
		t.Fatalf("deprecated HasMany failed: %v", err)
	}
	if posts == nil || len(*posts) != 1 {
		t.Fatalf("expected 1 post, got %v", posts)
	}

	author, err := BelongsTo[relPost, relUser](ctx,
		Query[relPost](db, postTable).WhereEq("title", "Hello"),
		userTable, "user_id")
	if err != nil {
		t.Fatalf("deprecated BelongsTo failed: %v", err)
	}
	if author == nil || author.Name != "Alice" {
		t.Fatalf("expected author Alice, got %+v", author)
	}
}
