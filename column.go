package surrealgoorm

type Column struct {
	Name         string
	Type         string
	IsUnique     bool
	DefaultValue any
	Optional     bool
}

func (c *Column) Unique() *Column {
	c.IsUnique = true
	return c
}
func (c *Column) Default(v any) *Column {
	c.DefaultValue = v
	return c
}
func (c *Column) Nullable() *Column {
	c.Optional = true
	return c
}
