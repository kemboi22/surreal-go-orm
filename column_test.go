package surrealgoorm

import "testing"

func TestColumnUnique(t *testing.T) {
	c := &Column{Name: "email", Type: "string"}
	result := c.Unique()
	if !c.IsUnique {
		t.Error("Unique() should set IsUnique to true")
	}
	if result != c {
		t.Error("Unique() should return the same column for chaining")
	}
}

func TestColumnDefault(t *testing.T) {
	c := &Column{Name: "age", Type: "int"}
	result := c.Default(18)
	if c.DefaultValue != 18 {
		t.Errorf("Default() should set DefaultValue, got %v", c.DefaultValue)
	}
	if result != c {
		t.Error("Default() should return the same column for chaining")
	}
}

func TestColumnDefaultString(t *testing.T) {
	c := &Column{Name: "name", Type: "string"}
	c.Default("active")
	if c.DefaultValue != "active" {
		t.Errorf("Default() should set string value, got %v", c.DefaultValue)
	}
}

func TestColumnNullable(t *testing.T) {
	c := &Column{Name: "name", Type: "string"}
	result := c.Nullable()
	if !c.Optional {
		t.Error("Nullable() should set Optional to true")
	}
	if result != c {
		t.Error("Nullable() should return the same column for chaining")
	}
}

func TestColumnChaining(t *testing.T) {
	c := &Column{Name: "email", Type: "string"}
	c.Unique().Nullable().Default("test@example.com")
	if !c.IsUnique {
		t.Error("Unique() should set IsUnique via chaining")
	}
	if !c.Optional {
		t.Error("Nullable() should set Optional via chaining")
	}
	if c.DefaultValue != "test@example.com" {
		t.Error("Default() should set DefaultValue via chaining")
	}
}

func TestColumnIndependentModifications(t *testing.T) {
	c1 := &Column{Name: "a", Type: "string"}
	c2 := &Column{Name: "b", Type: "int"}

	c1.Unique()
	c2.Nullable().Default(0)

	if !c1.IsUnique || c2.IsUnique {
		t.Error("Unique should only affect the target column")
	}
	if c1.Optional || !c2.Optional {
		t.Error("Nullable should only affect the target column")
	}
	if c1.DefaultValue != nil || c2.DefaultValue != 0 {
		t.Error("Default should only affect the target column")
	}
}
