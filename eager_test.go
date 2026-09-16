package surrealgoorm

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/surrealdb/surrealdb.go/pkg/models"
)

type eagerPost struct {
	ID     string     `json:"id,omitempty"`
	Title  string     `json:"title"`
	UserID string     `json:"user_id"`
	User   *eagerUser `json:"user,omitempty" orm:"belongs_to,table:user,fk:user_id"`
}

type eagerUser struct {
	ID    string      `json:"id,omitempty"`
	Name  string      `json:"name"`
	Posts []eagerPost `json:"posts,omitempty" orm:"has_many,table:post,fk:user_id"`
}

type eagerUserFetch struct {
	ID    string      `json:"id,omitempty"`
	Name  string      `json:"name"`
	Posts []eagerPost `json:"posts,omitempty" orm:"fetch"`
}

type ambiguousUser struct {
	ID     string      `json:"id,omitempty"`
	Posts  []eagerPost `json:"posts,omitempty" orm:"has_many,table:post,fk:user_id"`
	Drafts []eagerPost `json:"drafts,omitempty" orm:"has_many,table:post,fk:user_id"`
}

func TestReflectMetaRelations(t *testing.T) {
	meta := reflectMeta(reflect.TypeOf(eagerUser{}))
	if len(meta.relations) != 1 {
		t.Fatalf("expected 1 relation, got %d", len(meta.relations))
	}
	rel := meta.relations[0]
	if rel.kind != relHasMany || rel.table != "post" || rel.fk != "user_id" {
		t.Fatalf("unexpected relation meta: %+v", rel)
	}
	if rel.fieldType != reflect.TypeOf([]eagerPost{}) {
		t.Fatalf("unexpected field type %s", rel.fieldType)
	}
	if !meta.relJSON["posts"] {
		t.Error("posts should be marked as relation json field")
	}
}

func TestStructToMapSkipsRelations(t *testing.T) {
	u := eagerUser{
		Name:  "Alice",
		Posts: []eagerPost{{Title: "Hello"}},
	}
	attrs, err := structToMap(&u)
	if err != nil {
		t.Fatalf("structToMap failed: %v", err)
	}
	if _, ok := attrs["posts"]; ok {
		t.Error("relation field posts should be excluded from structToMap")
	}
	if attrs["name"] != "Alice" {
		t.Errorf("expected name Alice, got %v", attrs["name"])
	}
}

func TestWithResolvesByType(t *testing.T) {
	q := Query[eagerUser](nil, "user").With[[]eagerPost]()
	if q.state.err != nil {
		t.Fatalf("With[[]eagerPost] failed: %v", q.state.err)
	}
	if len(q.state.with) != 1 || q.state.with[0].fieldName != "Posts" {
		t.Fatalf("expected Posts queued, got %+v", q.state.with)
	}
}

func TestWithAmbiguousRequiresField(t *testing.T) {
	q := Query[ambiguousUser](nil, "user").With[[]eagerPost]()
	if q.state.err == nil {
		t.Fatal("expected ambiguity error")
	}
	if !strings.Contains(q.state.err.Error(), "ambiguous") {
		t.Errorf("unexpected error: %v", q.state.err)
	}

	q2 := Query[ambiguousUser](nil, "user").WithField[[]eagerPost]("Drafts")
	if q2.state.err != nil {
		t.Fatalf("WithField failed: %v", q2.state.err)
	}
	if len(q2.state.with) != 1 || q2.state.with[0].fieldName != "Drafts" {
		t.Fatalf("expected Drafts queued, got %+v", q2.state.with)
	}
}

func TestWithFetchAddsClause(t *testing.T) {
	q := Query[eagerUserFetch](nil, "user").With[[]eagerPost]()
	if q.state.err != nil {
		t.Fatalf("With fetch failed: %v", q.state.err)
	}
	sql := q.ToSQL()
	if !strings.Contains(strings.ToUpper(sql), "FETCH") {
		t.Errorf("expected FETCH in SQL, got: %s", sql)
	}
}

func TestWithNamesUnknownRelation(t *testing.T) {
	q := Query[eagerUser](nil, "user").WithNames("missing")
	if q.state.err == nil {
		t.Fatal("expected error for unknown relation")
	}
}

type eagerProfile struct {
	ID     string `json:"id,omitempty"`
	Bio    string `json:"bio"`
	UserID string `json:"user_id"`
}

type eagerUserWithProfile struct {
	ID      string        `json:"id,omitempty"`
	Name    string        `json:"name"`
	Posts   []eagerPost   `json:"posts,omitempty" orm:"has_many,table:post,fk:user_id"`
	Profile *eagerProfile `json:"profile,omitempty" orm:"has_one,table:profile,fk:user_id"`
}

type incompleteRelUser struct {
	ID    string      `json:"id,omitempty"`
	Posts []eagerPost `json:"posts,omitempty" orm:"has_many"`
}

func TestWithUnknownType(t *testing.T) {
	q := Query[eagerUser](nil, "user").With[*eagerPost]()
	if q.state.err == nil {
		t.Fatal("expected error for unmatched relation type")
	}
	if !strings.Contains(q.state.err.Error(), "no relation field of type") {
		t.Errorf("unexpected error: %v", q.state.err)
	}
}

func TestWithFieldTypeMismatch(t *testing.T) {
	q := Query[eagerUser](nil, "user").WithField[*eagerPost]("Posts")
	if q.state.err == nil {
		t.Fatal("expected type mismatch error")
	}
	if !strings.Contains(q.state.err.Error(), "has type") {
		t.Errorf("unexpected error: %v", q.state.err)
	}
}

func TestWithFieldByJSONName(t *testing.T) {
	q := Query[eagerUser](nil, "user").WithField[[]eagerPost]("posts")
	if q.state.err != nil {
		t.Fatalf("WithField by json name failed: %v", q.state.err)
	}
	if len(q.state.with) != 1 || q.state.with[0].fieldName != "Posts" {
		t.Fatalf("expected Posts queued, got %+v", q.state.with)
	}
}

func TestWithNamesResolvesGoAndJSON(t *testing.T) {
	q := Query[eagerUser](nil, "user").WithNames("Posts")
	if q.state.err != nil {
		t.Fatalf("WithNames(Posts) failed: %v", q.state.err)
	}
	q2 := Query[eagerUser](nil, "user").WithNames("posts")
	if q2.state.err != nil {
		t.Fatalf("WithNames(posts) failed: %v", q2.state.err)
	}
}

func TestWithRequiresTableAndFK(t *testing.T) {
	q := Query[incompleteRelUser](nil, "user").With[[]eagerPost]()
	if q.state.err == nil {
		t.Fatal("expected error when has_many is missing table/fk")
	}
	if !strings.Contains(q.state.err.Error(), "requires table and fk") {
		t.Errorf("unexpected error: %v", q.state.err)
	}
}

func TestWithChainsDistinctTypes(t *testing.T) {
	q := Query[eagerUserWithProfile](nil, "user").
		With[[]eagerPost]().
		With[*eagerProfile]()
	if q.state.err != nil {
		t.Fatalf("chained With failed: %v", q.state.err)
	}
	if len(q.state.with) != 2 {
		t.Fatalf("expected 2 relations, got %d", len(q.state.with))
	}
}

func TestWithIsIdempotent(t *testing.T) {
	q := Query[eagerUser](nil, "user").With[[]eagerPost]().WithNames("Posts")
	if q.state.err != nil {
		t.Fatalf("duplicate With failed: %v", q.state.err)
	}
	if len(q.state.with) != 1 {
		t.Fatalf("expected 1 queued relation, got %d", len(q.state.with))
	}
}

func TestGetSurfacesWithError(t *testing.T) {
	_, err := Query[eagerUser](nil, "user").With[*eagerPost]().Get(context.Background())
	if err == nil {
		t.Fatal("expected Get to return the With resolution error")
	}
}

func TestSetRelationFieldHasMany(t *testing.T) {
	u := eagerUser{Name: "Alice"}
	rel := reflectMeta(reflect.TypeOf(eagerUser{})).relations[0]
	items := []reflect.Value{
		reflect.ValueOf(eagerPost{Title: "Hello"}),
		reflect.ValueOf(eagerPost{Title: "World"}),
	}
	if err := setRelationField(&u, rel, items); err != nil {
		t.Fatalf("setRelationField failed: %v", err)
	}
	if len(u.Posts) != 2 || u.Posts[0].Title != "Hello" || u.Posts[1].Title != "World" {
		t.Fatalf("unexpected posts: %+v", u.Posts)
	}
}

func TestSetRelationFieldBelongsToPointer(t *testing.T) {
	p := eagerPost{Title: "Hello"}
	rel := reflectMeta(reflect.TypeOf(eagerPost{})).relations[0]
	item := reflect.ValueOf(eagerUser{Name: "Alice"})
	if err := setRelationField(&p, rel, []reflect.Value{item}); err != nil {
		t.Fatalf("setRelationField failed: %v", err)
	}
	if p.User == nil || p.User.Name != "Alice" {
		t.Fatalf("expected nested user Alice, got %+v", p.User)
	}
}

func TestSetRelationFieldHasOneEmpty(t *testing.T) {
	u := eagerUserWithProfile{Profile: &eagerProfile{Bio: "stale"}}
	rel := reflectMeta(reflect.TypeOf(eagerUserWithProfile{})).relByName["Profile"]
	if err := setRelationField(&u, rel, nil); err != nil {
		t.Fatalf("setRelationField failed: %v", err)
	}
	if u.Profile != nil {
		t.Fatalf("expected nil profile, got %+v", u.Profile)
	}
}

func TestIDKeyNormalizesRecordID(t *testing.T) {
	rid := models.NewRecordID("user", "alice")
	if idKey(rid) != rid.String() || idKey(&rid) != rid.String() {
		t.Fatalf("idKey should stringify RecordID, got %q / %q", idKey(rid), idKey(&rid))
	}
	if idKey("user:alice") != rid.String() {
		t.Errorf("string id %q should match RecordID %q", idKey("user:alice"), rid.String())
	}
}
