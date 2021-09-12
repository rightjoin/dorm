package dorm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAttributeBool(t *testing.T) {

	var val interface{}
	var err error

	a := Attribute{
		Datatype: "bool",
	}

	// Truthy
	val, err = a.Accepts("true")
	assert.Equal(t, true, val)
	assert.Nil(t, err)

	val, err = a.Accepts("yes")
	assert.Equal(t, true, val)
	assert.Nil(t, err)

	val, err = a.Accepts("Y")
	assert.Equal(t, true, val)
	assert.Nil(t, err)

	val, err = a.Accepts("1")
	assert.Equal(t, true, val)
	assert.Nil(t, err)

	// Falsy
	val, err = a.Accepts("false")
	assert.Equal(t, false, val)
	assert.Nil(t, err)

	val, err = a.Accepts("No")
	assert.Equal(t, false, val)
	assert.Nil(t, err)

	val, err = a.Accepts("N")
	assert.Equal(t, false, val)
	assert.Nil(t, err)

	val, err = a.Accepts("0")
	assert.Equal(t, false, val)
	assert.Nil(t, err)

	// Incorrect input
	val, err = a.Accepts("123")
	assert.NotNil(t, err)
}

func TestAttributeInt(t *testing.T) {

	var val interface{}
	var err error

	a := Attribute{
		Datatype: "int",
	}

	// Truthy
	val, err = a.Accepts("12345")
	assert.Equal(t, 12345, val)
	assert.Nil(t, err)

	// Incorrect input
	val, err = a.Accepts("abc")
	assert.NotNil(t, err)
}

func TestAttributeDecimal(t *testing.T) {

	var val interface{}
	var err error

	a := Attribute{
		Datatype: "decimal",
	}

	// Truthy
	val, err = a.Accepts("123.45")
	assert.Equal(t, 123.45, val)
	assert.Nil(t, err)

	// Incorrect input
	val, err = a.Accepts("abc")
	assert.NotNil(t, err)
}

func TestAttributeString(t *testing.T) {

	var val interface{}
	var err error

	a := Attribute{
		Datatype: "string",
	}

	// Truthy
	val, err = a.Accepts("whassup")
	assert.Equal(t, "whassup", val)
	assert.Nil(t, err)

	val, err = a.Accepts("123")
	assert.NotEqual(t, 123, val)
}

func TestAttributeSuperset(t *testing.T) {

	var val interface{}
	var err error

	// String superset
	stry := Attribute{
		Datatype: "string",
	}

	val, err = stry.Accepts("a")
	assert.Equal(t, "a", val)
	assert.Nil(t, err)

	val, err = stry.Accepts("d")
	assert.Nil(t, err)

	// Int superset
	inty := Attribute{
		Datatype: "int",
	}

	val, err = inty.Accepts("123")
	assert.Equal(t, 123, val)
	assert.Nil(t, err)

	val, err = inty.Accepts("901")
	assert.Nil(t, err)

	// Decimal superset
	decy := Attribute{
		Datatype: "decimal",
	}

	val, err = decy.Accepts("1.23")
	assert.Equal(t, 1.23, val)
	assert.Nil(t, err)

	val, err = decy.Accepts("9.01")
	assert.Nil(t, err)
}

func TestAttributeUnits(t *testing.T) {

	var val interface{}
	var err error

	stry := Attribute{
		Datatype: "string",
		Units:    NewJArrStr("km", "mm"),
	}

	// Truthy
	val, err = stry.Accepts("1 km")
	assert.Equal(t, "1 km", val)
	assert.Nil(t, err)

	val, err = stry.Accepts("1.25 mm")
	assert.Equal(t, "1.25 mm", val)
	assert.Nil(t, err)

	// Error
	val, err = stry.Accepts("5 m")
	assert.NotNil(t, err)

}

func TestAttributeEnumMultiSelect(t *testing.T) {
	var val interface{}
	var err error

	var multiSelect uint8 = 1

	enumStrSelecty := Attribute{
		Datatype:    "string",
		MultiSelect: &multiSelect,
		Enums:       NewJArr("a", "b", "c"),
	}

	// Must Pass: all the values
	val, err = enumStrSelecty.Accepts(`["a","b","c"]`)
	assert.Nil(t, err)
	assert.Equal(t, []interface{}{"a", "b", "c"}, val)

	// Must Pass: subset of correct values
	val, err = enumStrSelecty.Accepts(`["a","c"]`)
	assert.Nil(t, err)
	assert.Equal(t, []interface{}{"a", "c"}, val)

	// Must fail: contains an invalid value for the enum
	val, err = enumStrSelecty.Accepts(`["a","b","d"]`)
	assert.NotNil(t, err)

	// Test for other data types
	enumIntSelecty := Attribute{
		Datatype:    "int",
		Enums:       NewJArr(1, 2, 3),
		MultiSelect: &multiSelect,
	}

	// Must pass
	val, err = enumIntSelecty.Accepts(`[1,2,3]`)
	assert.Nil(t, err)
	assert.Equal(t, []interface{}{1, 2, 3}, val)

	// Must fail
	val, err = enumIntSelecty.Accepts(`[1,4,5]`)
	assert.NotNil(t, err)

	val, err = enumIntSelecty.Accepts(`[1,2,"a"]`)
	assert.NotNil(t, err)
}
