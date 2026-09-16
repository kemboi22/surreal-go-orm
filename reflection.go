package surrealgoorm

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/surrealdb/surrealdb.go/pkg/models"
)

type relKind int

const (
	relNone relKind = iota
	relHasMany
	relHasOne
	relBelongsTo
	relFetch
)

type relationMeta struct {
	kind      relKind
	fieldName string
	jsonName  string
	table     string
	fk        string
	fieldType reflect.Type
	elemType  reflect.Type // related model struct type
	index     int
}

type modelMeta struct {
	timestamps bool
	softDelete bool
	hasID      bool
	fillable   map[string]bool
	guarded    map[string]bool
	relations  []*relationMeta
	relByName  map[string]*relationMeta
	relJSON    map[string]bool
}

func reflectMeta(t reflect.Type) *modelMeta {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	meta := &modelMeta{
		fillable:  map[string]bool{},
		guarded:   map[string]bool{},
		relByName: map[string]*relationMeta{},
		relJSON:   map[string]bool{},
	}
	if t.Kind() != reflect.Struct {
		return meta
	}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Anonymous && jsonFieldName(f) == "" {
			walkEmbedMeta(f.Type, meta)
			continue
		}
		name := jsonFieldName(f)
		if name == "" || name == "-" {
			continue
		}
		applyFieldMeta(name, f.Tag.Get("orm"), meta)
		kind, table, fk := parseORMRelation(f.Tag.Get("orm"))
		if kind == relNone {
			continue
		}
		rel := &relationMeta{
			kind:      kind,
			fieldName: f.Name,
			jsonName:  name,
			table:     table,
			fk:        fk,
			fieldType: f.Type,
			elemType:  relatedElemType(f.Type),
			index:     i,
		}
		meta.relations = append(meta.relations, rel)
		meta.relByName[f.Name] = rel
		meta.relByName[name] = rel
		meta.relJSON[name] = true
	}
	return meta
}

func walkEmbedMeta(rt reflect.Type, meta *modelMeta) {
	if rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}
	if rt.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if f.Anonymous && jsonFieldName(f) == "" {
			walkEmbedMeta(f.Type, meta)
			continue
		}
		name := jsonFieldName(f)
		if name == "" || name == "-" {
			continue
		}
		applyFieldMeta(name, f.Tag.Get("orm"), meta)
	}
}

func applyFieldMeta(name, ormTag string, meta *modelMeta) {
	switch name {
	case "id":
		meta.hasID = true
	case "created_at", "updated_at":
		meta.timestamps = true
	case "deleted_at":
		meta.softDelete = true
	}
	for _, part := range strings.Split(ormTag, ",") {
		switch strings.TrimSpace(part) {
		case "fillable":
			meta.fillable[name] = true
		case "guarded":
			meta.guarded[name] = true
		}
	}
}

func parseORMRelation(tag string) (kind relKind, table, fk string) {
	for _, part := range strings.Split(tag, ",") {
		part = strings.TrimSpace(part)
		switch {
		case part == "has_many":
			kind = relHasMany
		case part == "has_one":
			kind = relHasOne
		case part == "belongs_to":
			kind = relBelongsTo
		case part == "fetch":
			kind = relFetch
		case strings.HasPrefix(part, "table:"):
			table = strings.TrimPrefix(part, "table:")
		case strings.HasPrefix(part, "fk:"):
			fk = strings.TrimPrefix(part, "fk:")
		}
	}
	return kind, table, fk
}

func relatedElemType(ft reflect.Type) reflect.Type {
	if ft.Kind() == reflect.Slice {
		ft = ft.Elem()
	}
	if ft.Kind() == reflect.Pointer {
		ft = ft.Elem()
	}
	return ft
}

func (meta *modelMeta) relationByType(t reflect.Type) (*relationMeta, error) {
	if meta == nil {
		return nil, fmt.Errorf("surrealgoorm: no model metadata")
	}
	var matches []*relationMeta
	for _, rel := range meta.relations {
		if rel.fieldType == t {
			matches = append(matches, rel)
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("surrealgoorm: no relation field of type %s", t)
	case 1:
		return matches[0], nil
	default:
		names := make([]string, len(matches))
		for i, rel := range matches {
			names[i] = rel.fieldName
		}
		return nil, fmt.Errorf("surrealgoorm: ambiguous relation type %s (fields %s); use WithField", t, strings.Join(names, ", "))
	}
}

func (meta *modelMeta) relationByName(name string) (*relationMeta, error) {
	if meta == nil {
		return nil, fmt.Errorf("surrealgoorm: no model metadata")
	}
	rel, ok := meta.relByName[name]
	if !ok {
		return nil, fmt.Errorf("surrealgoorm: unknown relation %q", name)
	}
	return rel, nil
}

func jsonFieldName(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	if tag == "" {
		return ""
	}
	return strings.Split(tag, ",")[0]
}

func walkFields(rv reflect.Value, fn func(name string, fv reflect.Value)) {
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return
	}
	t := rv.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Anonymous && jsonFieldName(f) == "" {
			walkFields(rv.Field(i), fn)
			continue
		}
		name := jsonFieldName(f)
		if name == "" || name == "-" {
			continue
		}
		fn(name, rv.Field(i))
	}
}

func structToMap(v any) (map[string]any, error) {
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil, fmt.Errorf("structToMap: value is nil")
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("structToMap: expected struct, got %T", v)
	}
	out := map[string]any{}
	meta := reflectMeta(rv.Type())
	walkFields(rv, func(name string, fv reflect.Value) {
		if meta.relJSON[name] {
			return
		}
		if fv.Kind() == reflect.Pointer {
			if fv.IsNil() {
				return
			}
			fv = fv.Elem()
		}
		out[name] = fv.Interface()
	})
	return out, nil
}

func mapToStruct(raw map[string]any, out any) error {
	rv := reflect.ValueOf(out)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("mapToStruct: out must be a non-nil pointer, got %T", out)
	}
	for rv.Elem().Kind() == reflect.Pointer {
		e := rv.Elem()
		if e.IsNil() {
			e.Set(reflect.New(e.Type().Elem()))
		}
		rv = e
	}
	target := rv.Elem()
	if target.Kind() != reflect.Struct {
		return fmt.Errorf("mapToStruct: target must be a struct, got %s", target.Kind())
	}

	idVal, hasID := raw["id"]
	body := make(map[string]any, len(raw))
	for k, v := range raw {
		if k != "id" {
			body[k] = v
		}
	}
	b, err := json.Marshal(sanitizeJSON(body))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, target.Addr().Interface()); err != nil {
		return err
	}
	if hasID {
		setID(target.Addr().Interface(), idVal)
	}
	return nil
}

func sanitizeJSON(v any) any {
	switch t := v.(type) {
	case nil:
		return nil
	case models.RecordID:
		return t.String()
	case *models.RecordID:
		return t.String()
	case time.Time:
		return t
	case models.CustomDateTime:
		return t.Time
	case *models.CustomDateTime:
		return t.Time
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Map:
		out := map[string]any{}
		iter := rv.MapRange()
		for iter.Next() {
			out[fmt.Sprintf("%v", iter.Key().Interface())] = sanitizeJSON(iter.Value().Interface())
		}
		return out
	case reflect.Slice, reflect.Array:
		out := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			out[i] = sanitizeJSON(rv.Index(i).Interface())
		}
		return out
	default:
		return v
	}
}

func setID(out any, idVal any) {
	switch v := idVal.(type) {
	case models.RecordID:
		setIDField(out, &v)
	case *models.RecordID:
		setIDField(out, v)
	default:
		setField(out, "id", idVal)
	}
}

func setIDField(out any, rid *models.RecordID) {
	rv := reflect.ValueOf(out)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return
		}
		rv = rv.Elem()
	}
	walkFields(rv, func(name string, fv reflect.Value) {
		if name != "id" {
			return
		}
		switch {
		case fv.Kind() == reflect.String && fv.CanSet():
			fv.SetString(rid.String())
		case fv.Type() == reflect.TypeOf(models.RecordID{}):
			fv.Set(reflect.ValueOf(*rid))
		case fv.Type() == reflect.TypeOf(&models.RecordID{}):
			fv.Set(reflect.ValueOf(rid))
		}
	})
}

func indirect(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}
	return v
}

func setField(out any, name string, value any) {
	rv := reflect.ValueOf(out)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return
		}
		rv = rv.Elem()
	}
	walkFields(rv, func(n string, fv reflect.Value) {
		if n != name {
			return
		}
		if !fv.CanSet() {
			return
		}
		fv = indirect(fv)
		switch fv.Kind() {
		case reflect.String:
			fv.SetString(fmt.Sprintf("%v", value))
		case reflect.Struct:
			if fv.Type() == reflect.TypeOf(time.Time{}) {
				if t, ok := value.(time.Time); ok {
					fv.Set(reflect.ValueOf(t))
				}
			}
		case reflect.Bool:
			if b, ok := value.(bool); ok {
				fv.SetBool(b)
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if n, ok := toInt64(value); ok {
				fv.SetInt(n)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if n, ok := toInt64(value); ok {
				fv.SetUint(uint64(n))
			}
		case reflect.Float32, reflect.Float64:
			if f, ok := toFloat64(value); ok {
				fv.SetFloat(f)
			}
		}
	})
}

func applyMapToStruct(out any, attrs map[string]any) error {
	for k, v := range attrs {
		setField(out, k, v)
	}
	return nil
}

func applyTimestamps(attrs map[string]any, data any, creating bool, meta *modelMeta) {
	if meta == nil || !meta.timestamps {
		return
	}
	now := time.Now()
	if creating {
		if !hasTime(attrs["created_at"]) {
			attrs["created_at"] = now
			setTimestampField(data, "created_at", now)
		}
	}
	if !hasTime(attrs["updated_at"]) {
		attrs["updated_at"] = now
		setTimestampField(data, "updated_at", now)
	}
}

func hasTime(v any) bool {
	if v == nil {
		return false
	}
	switch t := v.(type) {
	case time.Time:
		return !t.IsZero()
	case *time.Time:
		return t != nil && !t.IsZero()
	case string:
		return t != ""
	case models.CustomDateTime:
		return !t.Time.IsZero()
	case *models.CustomDateTime:
		return t != nil && !t.Time.IsZero()
	}
	return false
}

func setTimestampField(data any, name string, now time.Time) {
	if data == nil {
		return
	}
	rv := reflect.ValueOf(data)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return
		}
		rv = rv.Elem()
	}
	walkFields(rv, func(n string, fv reflect.Value) {
		if n != name || !fv.CanSet() {
			return
		}
		fv = indirect(fv)
		switch fv.Kind() {
		case reflect.String:
			fv.SetString(now.Format(time.RFC3339))
		case reflect.Struct:
			if fv.Type() == reflect.TypeOf(time.Time{}) {
				fv.Set(reflect.ValueOf(now))
			}
		}
	})
}

func getID(v any) (any, bool) {
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil, false
		}
		rv = rv.Elem()
	}
	var found any
	walkFields(rv, func(name string, fv reflect.Value) {
		if name != "id" {
			return
		}
		fv = indirect(fv)
		if fv.IsValid() && !fv.IsZero() {
			found = fv.Interface()
		}
	})
	return found, found != nil
}

func cloneMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func toInt(v any) int {
	if n, ok := toInt64(v); ok {
		return int(n)
	}
	if f, ok := toFloat64(v); ok {
		return int(f)
	}
	return 0
}

func toInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int8:
		return int64(n), true
	case int16:
		return int64(n), true
	case int32:
		return int64(n), true
	case int64:
		return n, true
	case uint:
		return int64(n), true
	case uint64:
		return int64(n), true
	case float32:
		return int64(n), true
	case float64:
		return int64(n), true
	case json.Number:
		i, err := n.Int64()
		return i, err == nil
	default:
		return 0, false
	}
}

func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}
