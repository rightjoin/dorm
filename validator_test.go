package dorm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMustInsert(t *testing.T) {
	type Abc struct {
		Alphabet string `insert:"must"`
		Numbers  string `insert:"no"`
	}

	// must-insert field missing
	ok, _ := validateModel(&Abc{}, map[string]string{"some": "thing"}, "insert")
	assert.False(t, ok)

	// must-insert present
	ok, _ = validateModel(&Abc{}, map[string]string{"alphabet": "abcdef"}, "insert")
	assert.True(t, ok)

	// do-not-insert fields given
	ok, _ = validateModel(&Abc{}, map[string]string{"alphabet": "abcdef", "numbers": "12345"}, "insert")
	assert.False(t, ok)
}

func TestMustUpdate(t *testing.T) {
	type Abc struct {
		Alphabet string `update:"must"`
		Numbers  string `update:"no"`
	}

	// must-update field missing
	ok, _ := validateModel(&Abc{}, map[string]string{"some": "thing"}, "update")
	assert.False(t, ok)

	// must-update present
	ok, _ = validateModel(&Abc{}, map[string]string{"alphabet": "abcdef"}, "update")
	assert.True(t, ok)

	// do-not-update fields given
	ok, _ = validateModel(&Abc{}, map[string]string{"alphabet": "abcdef", "numbers": "12345"}, "update")
	assert.False(t, ok)
}

func TestTrimming(t *testing.T) {
	type TrimMe struct {
		Str   string
		Pstr  *string
		Str2  string  `trim:"no"`
		Pstr2 *string `trim:"no"`
	}

	input := map[string]string{
		"str":   " str ",
		"pstr":  " pstr ",
		"str2":  " str2 ",
		"pstr2": " pstr2 ",
	}

	ok, _ := validateModel(&TrimMe{}, input, "insert")
	assert.True(t, ok)

	assert.Equal(t, "str", input["str"])
	assert.Equal(t, "pstr", input["pstr"])
	assert.Equal(t, " str2 ", input["str2"])
	assert.Equal(t, " pstr2 ", input["pstr2"])

}

func TestValidate(t *testing.T) {
	type Abc struct {
		Alphabet string `validate:"email"`
	}

	ok, _ := validateModel(&Abc{}, map[string]string{"alphabet": "any@gmail.com"}, "insert")
	assert.True(t, ok)

	ok, _ = validateModel(&Abc{}, map[string]string{"alphabet": "any@gmail"}, "insert")
	assert.False(t, ok)
}

func TestJson(t *testing.T) {
	type Abc struct {
		Arr    JArr
		PtrArr *JArr

		Doc    JDoc
		PtrDoc *JDoc
	}

	// array validations:

	// passing string is a problem
	ok, _ := validateModel(&Abc{}, map[string]string{"ptr_arr": "stringy"}, "insert")
	assert.False(t, ok)
	ok, _ = validateModel(&Abc{}, map[string]string{"ptr_arr": "stringy"}, "update")
	assert.False(t, ok)

	// passing array of int
	ok, _ = validateModel(&Abc{}, map[string]string{"arr": "[1,2,3]"}, "insert")
	assert.True(t, ok)
	ok, _ = validateModel(&Abc{}, map[string]string{"ptr_arr": "[1,2,3]"}, "update")
	assert.True(t, ok)

	// document validations:

	// passing string is a problem
	ok, _ = validateModel(&Abc{}, map[string]string{"doc": "stringy"}, "insert")
	assert.False(t, ok)
	ok, _ = validateModel(&Abc{}, map[string]string{"ptr_doc": "stringy"}, "update")
	assert.False(t, ok)

	// passing array is a problem
	ok, _ = validateModel(&Abc{}, map[string]string{"doc": "[1,2,3]"}, "insert")
	assert.False(t, ok)
	ok, _ = validateModel(&Abc{}, map[string]string{"ptr_doc": "[1,2,3]"}, "update")
	assert.False(t, ok)

	// passing document
	ok, _ = validateModel(&Abc{}, map[string]string{"doc": `{"key":"value"}`}, "insert")
	assert.True(t, ok)
	ok, _ = validateModel(&Abc{}, map[string]string{"ptr_doc": `{"key":"value"}`}, "update")
	assert.True(t, ok)
}
