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
	val, err = a.accetps("true")
	assert.Equal(t, true, val)
	assert.Nil(t, err)

	val, err = a.accetps("yes")
	assert.Equal(t, true, val)
	assert.Nil(t, err)

	val, err = a.accetps("Y")
	assert.Equal(t, true, val)
	assert.Nil(t, err)

	val, err = a.accetps("1")
	assert.Equal(t, true, val)
	assert.Nil(t, err)

	// Falsy
	val, err = a.accetps("false")
	assert.Equal(t, false, val)
	assert.Nil(t, err)

	val, err = a.accetps("No")
	assert.Equal(t, false, val)
	assert.Nil(t, err)

	val, err = a.accetps("N")
	assert.Equal(t, false, val)
	assert.Nil(t, err)

	val, err = a.accetps("0")
	assert.Equal(t, false, val)
	assert.Nil(t, err)

	// Incorrect input
	val, err = a.accetps("123")
	assert.NotNil(t, err)
}

func TestAttributeInt(t *testing.T) {

	var val interface{}
	var err error

	a := Attribute{
		Datatype: "int",
	}

	// Truthy
	val, err = a.accetps("12345")
	assert.Equal(t, 12345, val)
	assert.Nil(t, err)

	// Incorrect input
	val, err = a.accetps("abc")
	assert.NotNil(t, err)
}

func TestAttributeDecimal(t *testing.T) {

	var val interface{}
	var err error

	a := Attribute{
		Datatype: "decimal",
	}

	// Truthy
	val, err = a.accetps("123.45")
	assert.Equal(t, 123.45, val)
	assert.Nil(t, err)

	// Incorrect input
	val, err = a.accetps("abc")
	assert.NotNil(t, err)
}

func TestAttributeString(t *testing.T) {

	var val interface{}
	var err error

	a := Attribute{
		Datatype: "string",
	}

	// Truthy
	val, err = a.accetps("whassup")
	assert.Equal(t, "whassup", val)
	assert.Nil(t, err)

	val, err = a.accetps("123")
	assert.NotEqual(t, 123, val)
}

func TestAttributeSuperset(t *testing.T) {

	var val interface{}
	var err error

	// String superset
	stry := Attribute{
		Datatype: "string",
		Superset: NewJArr("a", "b", "c"),
	}

	val, err = stry.accetps("a")
	assert.Equal(t, "a", val)
	assert.Nil(t, err)

	val, err = stry.accetps("d")
	assert.NotNil(t, err)

	// Int superset
	inty := Attribute{
		Datatype: "int",
		Superset: NewJArr(123, 456, 789),
	}

	val, err = inty.accetps("123")
	assert.Equal(t, 123, val)
	assert.Nil(t, err)

	val, err = inty.accetps("901")
	assert.NotNil(t, err)

	// Decimal superset
	decy := Attribute{
		Datatype: "decimal",
		Superset: NewJArr(1.23, 4.56, 7.89),
	}

	val, err = decy.accetps("1.23")
	assert.Equal(t, 1.23, val)
	assert.Nil(t, err)

	val, err = decy.accetps("9.01")
	assert.NotNil(t, err)
}

func TestAttributeUnits(t *testing.T) {

	var val interface{}
	var err error

	stry := Attribute{
		Datatype: "string",
		Units:    NewJArrStr("km", "mm"),
	}

	// Truthy
	val, err = stry.accetps("1 km")
	assert.Equal(t, "1 km", val)
	assert.Nil(t, err)

	val, err = stry.accetps("1.25 mm")
	assert.Equal(t, "1.25 mm", val)
	assert.Nil(t, err)

	// Error
	val, err = stry.accetps("5 m")
	assert.NotNil(t, err)

}
