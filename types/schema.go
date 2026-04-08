package types

type FieldType string

const (
	FieldTypeString   FieldType = "string"
	FieldTypeInt      FieldType = "int"
	FieldTypeFloat    FieldType = "float"
	FieldTypeBool     FieldType = "bool"
	FieldTypeDatetime FieldType = "datetime"
	FieldTypeText     FieldType = "text"
	FieldTypeUUID     FieldType = "uuid"
	FieldTypeArray    FieldType = "array"
	FieldTypeObject   FieldType = "object"
	FieldTypeRecord   FieldType = "record"
	FieldTypeNumber   FieldType = "number"
)

type FieldOption struct {
	Type          FieldType
	Default       any
	Unique        bool
	Nullable      bool
	Index         bool
	FullText      bool
	NotNull       bool
	AutoIncrement bool
}

type SchemaField struct {
	Name    string
	Type    FieldType
	Options []FieldOption
}

func String(name string) *ColumnBuilder {
	return &ColumnBuilder{
		name:      name,
		fieldType: FieldTypeString,
	}
}

func Integer(name string) *ColumnBuilder {
	return &ColumnBuilder{
		name:      name,
		fieldType: FieldTypeInt,
	}
}

func Float(name string) *ColumnBuilder {
	return &ColumnBuilder{
		name:      name,
		fieldType: FieldTypeFloat,
	}
}

func Boolean(name string) *ColumnBuilder {
	return &ColumnBuilder{
		name:      name,
		fieldType: FieldTypeBool,
	}
}

func Timestamp(name string) *ColumnBuilder {
	return &ColumnBuilder{
		name:      name,
		fieldType: FieldTypeDatetime,
	}
}

func Text(name string) *ColumnBuilder {
	return &ColumnBuilder{
		name:      name,
		fieldType: FieldTypeText,
	}
}

func UUID(name string) *ColumnBuilder {
	return &ColumnBuilder{
		name:      name,
		fieldType: FieldTypeUUID,
	}
}

func Array(name string) *ColumnBuilder {
	return &ColumnBuilder{
		name:      name,
		fieldType: FieldTypeArray,
	}
}

func Object(name string) *ColumnBuilder {
	return &ColumnBuilder{
		name:      name,
		fieldType: FieldTypeObject,
	}
}

type ColumnBuilder struct {
	name      string
	fieldType FieldType
	unique    bool
	nullable  bool
	default_  any
	index     bool
	notNull   bool
}

func (c *ColumnBuilder) Unique() *ColumnBuilder {
	c.unique = true
	return c
}

func (c *ColumnBuilder) Nullable() *ColumnBuilder {
	c.nullable = true
	return c
}

func (c *ColumnBuilder) Default(v any) *ColumnBuilder {
	c.default_ = v
	return c
}

func (c *ColumnBuilder) Index() *ColumnBuilder {
	c.index = true
	return c
}

func (c *ColumnBuilder) NotNull() *ColumnBuilder {
	c.notNull = true
	return c
}

func (c *ColumnBuilder) Build() SchemaField {
	opts := []FieldOption{
		{Type: c.fieldType},
	}
	if c.unique {
		opts = append(opts, FieldOption{Unique: true})
	}
	if c.nullable {
		opts = append(opts, FieldOption{Nullable: true})
	}
	if c.default_ != nil {
		opts = append(opts, FieldOption{Default: c.default_})
	}
	if c.index {
		opts = append(opts, FieldOption{Index: true})
	}
	return SchemaField{
		Name:    c.name,
		Type:    c.fieldType,
		Options: opts,
	}
}
