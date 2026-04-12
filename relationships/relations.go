package relationships

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"

	surrealgoorm "github.com/kemboi22/surreal-go-orm"
	"github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

type BelongsTo[Parent any] struct {
	foreignKey string
	ownerKey   string
}

func NewBelongsTo[Parent any](foreignKey, ownerKey string) *BelongsTo[Parent] {
	return &BelongsTo[Parent]{
		foreignKey: foreignKey,
		ownerKey:   ownerKey,
	}
}

func (b *BelongsTo[Parent]) Get(ctx context.Context, db *surrealgoorm.DB, model any) (*Parent, error) {
	result := new(Parent)

	foreignValue := getFieldValue(model, b.foreignKey)
	if foreignValue == nil {
		return nil, fmt.Errorf("foreign key %s not found", b.foreignKey)
	}

	var rid models.RecordID
	switch v := foreignValue.(type) {
	case models.RecordID:
		rid = v
	case string:
		if strings.Contains(v, ":") {
			parts := strings.SplitN(v, ":", 2)
			rid = models.NewRecordID(parts[0], parts[1])
		} else {
			rid = models.NewRecordID("", v)
		}
	default:
		return nil, fmt.Errorf("invalid foreign key type")
	}

	record, err := surrealdb.Select[Parent](ctx, db.Raw(), rid)
	if err != nil {
		return nil, fmt.Errorf("failed to get related model: %w", err)
	}
	if record == nil {
		return nil, fmt.Errorf("related model not found")
	}

	*result = *record
	return result, nil
}

func (b *BelongsTo[Parent]) Set(ctx context.Context, db *surrealgoorm.DB, model any, parent *Parent) error {
	parentID := getModelID(parent)
	if parentID == "" {
		return fmt.Errorf("parent model has no ID")
	}

	setFieldValue(model, b.foreignKey, models.NewRecordID("", parentID))
	return nil
}

type HasMany[Related any] struct {
	foreignKey string
	localKey   string
}

func NewHasMany[Related any](foreignKey, localKey string) *HasMany[Related] {
	return &HasMany[Related]{
		foreignKey: foreignKey,
		localKey:   localKey,
	}
}

func (h *HasMany[Related]) Get(ctx context.Context, db *surrealgoorm.DB, model any) ([]Related, error) {
	var results []Related

	localID := getModelID(model)
	if localID == "" {
		return nil, fmt.Errorf("model has no ID")
	}

	rid := models.NewRecordID("", localID)

	sql := fmt.Sprintf("SELECT * FROM %s WHERE %s = $id", getTableNameFromType[Related](), h.foreignKey)
	resp, err := surrealdb.Query[[]map[string]any](ctx, db.Raw(), sql, map[string]any{"id": rid.String()})
	if err != nil {
		return nil, err
	}

	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil {
		return results, nil
	}

	for _, row := range (*resp)[0].Result {
		r := new(Related)
		if err := mapToStructORM(row, r); err != nil {
			continue
		}
		results = append(results, *r)
	}

	return results, nil
}

func (h *HasMany[Related]) Create(ctx context.Context, db *surrealgoorm.DB, model any, related *Related) error {
	localID := getModelID(model)
	if localID == "" {
		return fmt.Errorf("model has no ID")
	}

	setFieldValue(related, h.foreignKey, models.NewRecordID("", localID))
	return surrealgoorm.Create(ctx, db, related)
}

func (h *HasMany[Related]) SaveMany(ctx context.Context, db *surrealgoorm.DB, model any, relatedList []Related) error {
	for i := range relatedList {
		if err := h.Create(ctx, db, model, &relatedList[i]); err != nil {
			return err
		}
	}
	return nil
}

type HasOne[Related any] struct {
	foreignKey string
	localKey   string
}

func NewHasOne[Related any](foreignKey, localKey string) *HasOne[Related] {
	return &HasOne[Related]{
		foreignKey: foreignKey,
		localKey:   localKey,
	}
}

func (h *HasOne[Related]) Get(ctx context.Context, db *surrealgoorm.DB, model any) (*Related, error) {
	result := new(Related)

	localID := getModelID(model)
	if localID == "" {
		return nil, fmt.Errorf("model has no ID")
	}

	rid := models.NewRecordID("", localID)

	sql := fmt.Sprintf("SELECT * FROM %s WHERE %s = $id LIMIT 1", getTableNameFromType[Related](), h.foreignKey)
	resp, err := surrealdb.Query[[]map[string]any](ctx, db.Raw(), sql, map[string]any{"id": rid.String()})
	if err != nil {
		return nil, err
	}

	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil || len((*resp)[0].Result) == 0 {
		return nil, fmt.Errorf("related model not found")
	}

	if err := mapToStructORM((*resp)[0].Result[0], result); err != nil {
		return nil, err
	}

	return result, nil
}

func (h *HasOne[Related]) Create(ctx context.Context, db *surrealgoorm.DB, model any, related *Related) error {
	localID := getModelID(model)
	if localID == "" {
		return fmt.Errorf("model has no ID")
	}

	setFieldValue(related, h.foreignKey, models.NewRecordID("", localID))
	return surrealgoorm.Create(ctx, db, related)
}

func (h *HasOne[Related]) Save(ctx context.Context, db *surrealgoorm.DB, model any, related *Related) error {
	return h.Create(ctx, db, model, related)
}

type ManyToMany[Related any] struct {
	pivotTable string
	foreignKey string
	relatedKey string
}

func NewManyToMany[Related any](pivotTable, foreignKey, relatedKey string) *ManyToMany[Related] {
	return &ManyToMany[Related]{
		pivotTable: pivotTable,
		foreignKey: foreignKey,
		relatedKey: relatedKey,
	}
}

func (m *ManyToMany[Related]) Get(ctx context.Context, db *surrealgoorm.DB, model any) ([]Related, error) {
	var results []Related

	localID := getModelID(model)
	if localID == "" {
		return nil, fmt.Errorf("model has no ID")
	}

	rid := models.NewRecordID("", localID)
	relatedTable := getTableNameFromType[Related]()

	sql := fmt.Sprintf(`
		SELECT * FROM %s 
		WHERE id IN (SELECT out FROM %s WHERE in = $id)
	`, relatedTable, m.pivotTable)

	resp, err := surrealdb.Query[[]map[string]any](ctx, db.Raw(), sql, map[string]any{"id": rid.String()})
	if err != nil {
		return nil, err
	}

	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil {
		return results, nil
	}

	for _, row := range (*resp)[0].Result {
		r := new(Related)
		if err := mapToStructORM(row, r); err != nil {
			continue
		}
		results = append(results, *r)
	}

	return results, nil
}

func (m *ManyToMany[Related]) Attach(ctx context.Context, db *surrealgoorm.DB, model any, relatedID string) error {
	localID := getModelID(model)
	if localID == "" {
		return fmt.Errorf("model has no ID")
	}

	localRID := models.NewRecordID("", localID)
	relatedRID := models.NewRecordID("", relatedID)

	_, err := surrealdb.InsertRelation[any](ctx, db.Raw(), &surrealdb.Relationship{
		In:       localRID,
		Out:      relatedRID,
		Relation: models.Table(m.pivotTable),
	})

	return err
}

func (m *ManyToMany[Related]) AttachMany(ctx context.Context, db *surrealgoorm.DB, model any, relatedIDs []string) error {
	for _, relatedID := range relatedIDs {
		if err := m.Attach(ctx, db, model, relatedID); err != nil {
			return err
		}
	}
	return nil
}

func (m *ManyToMany[Related]) Detach(ctx context.Context, db *surrealgoorm.DB, model any, relatedID string) error {
	localID := getModelID(model)
	if localID == "" {
		return fmt.Errorf("model has no ID")
	}

	localRID := models.NewRecordID("", localID)
	relatedRID := models.NewRecordID("", relatedID)

	sql := fmt.Sprintf(`
		DELETE FROM %s WHERE in = $in AND out = $out
	`, m.pivotTable)

	_, err := surrealdb.Query[any](ctx, db.Raw(), sql, map[string]any{
		"in":  localRID.String(),
		"out": relatedRID.String(),
	})

	return err
}

func (m *ManyToMany[Related]) DetachMany(ctx context.Context, db *surrealgoorm.DB, model any, relatedIDs []string) error {
	for _, relatedID := range relatedIDs {
		if err := m.Detach(ctx, db, model, relatedID); err != nil {
			return err
		}
	}
	return nil
}

func (m *ManyToMany[Related]) Sync(ctx context.Context, db *surrealgoorm.DB, model any, relatedIDs []string) error {
	current, err := m.Get(ctx, db, model)
	if err != nil {
		return err
	}

	currentIDs := make(map[string]bool)
	for _, r := range current {
		if id := getModelID(&r); id != "" {
			currentIDs[id] = true
		}
	}

	for _, id := range relatedIDs {
		if !currentIDs[id] {
			if err := m.Attach(ctx, db, model, id); err != nil {
				return err
			}
		}
	}

	for id := range currentIDs {
		found := slices.Contains(relatedIDs, id)
		if !found {
			if err := m.Detach(ctx, db, model, id); err != nil {
				return err
			}
		}
	}

	return nil
}

func getModelID(model any) string {
	if model == nil {
		return ""
	}

	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return ""
	}

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if field.Type() == reflect.TypeFor[models.RecordID]() {
			if rid, ok := field.Interface().(models.RecordID); ok {
				return rid.String()
			}
		}
		if field.Type() == reflect.TypeFor[string]() {
			if fieldName := v.Type().Field(i).Name; fieldName == "ID" {
				return field.Interface().(string)
			}
		}
	}

	return ""
}

func getFieldValue(model any, field string) any {
	if model == nil {
		return nil
	}

	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil
	}

	for i := 0; i < v.NumField(); i++ {
		if v.Type().Field(i).Name == field {
			return v.Field(i).Interface()
		}
	}

	return nil
}

func setFieldValue(model any, field string, value any) {
	if model == nil {
		return
	}

	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < v.NumField(); i++ {
		if v.Type().Field(i).Name == field && v.Field(i).CanSet() {
			v.Field(i).Set(reflect.ValueOf(value))
			return
		}
	}
}

func getTableNameFromType[T any]() string {
	var t T
	if tn, ok := any(&t).(surrealgoorm.TableName); ok {
		return tn.TableName()
	}
	return ""
}

func mapToStructORM(m map[string]any, s any) error {
	if s == nil {
		return fmt.Errorf("destination is nil")
	}

	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return fmt.Errorf("expected struct, got %s", v.Kind())
	}

	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("orm")
		if tag == "" {
			continue
		}

		columnName, ok := ormColumnName(tag)
		if !ok || columnName == "-" {
			continue
		}

		if val, ok := m[columnName]; ok {
			fieldVal := v.Field(i)
			if fieldVal.CanSet() && val != nil {
				fieldVal.Set(reflect.ValueOf(val))
			}
		}
	}

	return nil
}

func ormColumnName(tag string) (string, bool) {
	for _, part := range strings.Split(tag, ";") {
		part = strings.TrimSpace(part)
		if !strings.HasPrefix(part, "column:") {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(part, "column:"))
		if value == "" {
			return "", false
		}
		return value, true
	}
	return "", false
}

type MorphTo[Related any] struct {
	_         string
	idField   string
	typeField string
}

func NewMorphTo[Related any](idField, typeField string) *MorphTo[Related] {
	return &MorphTo[Related]{
		idField:   idField,
		typeField: typeField,
	}
}

func (m *MorphTo[Related]) Get(ctx context.Context, db *surrealgoorm.DB, model any) (*Related, error) {
	result := new(Related)

	idValue := getFieldValue(model, m.idField)
	typeValue := getFieldValue(model, m.typeField)

	if idValue == nil || typeValue == nil {
		return nil, fmt.Errorf("morph fields not found")
	}

	rid, ok := idValue.(models.RecordID)
	if !ok {
		return nil, fmt.Errorf("invalid morph id type")
	}

	record, err := surrealdb.Select[Related](ctx, db.Raw(), rid)
	if err != nil {
		return nil, fmt.Errorf("failed to get related model: %w", err)
	}
	if record == nil {
		return nil, fmt.Errorf("related model not found")
	}

	_ = typeValue
	*result = *record
	return result, nil
}

type MorphOne[Related any] struct {
	foreignKey string
	localKey   string
	morphType  string
}

func NewMorphOne[Related any](foreignKey, localKey, morphType string) *MorphOne[Related] {
	return &MorphOne[Related]{
		foreignKey: foreignKey,
		localKey:   localKey,
		morphType:  morphType,
	}
}

func (m *MorphOne[Related]) Get(ctx context.Context, db *surrealgoorm.DB, model any) (*Related, error) {
	result := new(Related)
	localID := getModelID(model)
	if localID == "" {
		return nil, fmt.Errorf("model has no ID")
	}

	sql := fmt.Sprintf("SELECT * FROM %s WHERE %s = $id AND type = $type LIMIT 1", getTableNameFromType[Related](), m.foreignKey)
	resp, err := surrealdb.Query[[]map[string]any](ctx, db.Raw(), sql, map[string]any{
		"id":   localID,
		"type": m.morphType,
	})
	if err != nil {
		return nil, err
	}
	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil || len((*resp)[0].Result) == 0 {
		return nil, fmt.Errorf("related model not found")
	}
	if err := mapToStructORM((*resp)[0].Result[0], result); err != nil {
		return nil, err
	}
	return result, nil
}

type MorphMany[Related any] struct {
	foreignKey string
	localKey   string
	morphType  string
}

func NewMorphMany[Related any](foreignKey, localKey, morphType string) *MorphMany[Related] {
	return &MorphMany[Related]{
		foreignKey: foreignKey,
		localKey:   localKey,
		morphType:  morphType,
	}
}

func (m *MorphMany[Related]) Get(ctx context.Context, db *surrealgoorm.DB, model any) ([]Related, error) {
	localID := getModelID(model)
	if localID == "" {
		return nil, fmt.Errorf("model has no ID")
	}

	sql := fmt.Sprintf("SELECT * FROM %s WHERE %s = $id AND type = $type", getTableNameFromType[Related](), m.foreignKey)
	resp, err := surrealdb.Query[[]map[string]any](ctx, db.Raw(), sql, map[string]any{
		"id":   localID,
		"type": m.morphType,
	})
	if err != nil {
		return nil, err
	}
	if resp == nil || len(*resp) == 0 || (*resp)[0].Result == nil {
		return []Related{}, nil
	}

	results := make([]Related, 0, len((*resp)[0].Result))
	for _, row := range (*resp)[0].Result {
		item := new(Related)
		if err := mapToStructORM(row, item); err != nil {
			return nil, err
		}
		results = append(results, *item)
	}
	return results, nil
}
