package dorm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildInsertSql(t *testing.T) {
	var vars = map[string]string{
		"field": "value",
	}

	sql, params := buildInsertSql("table_name", vars)

	assert.Equal(t, "INSERT INTO `table_name` (`field`) VALUES (?)", sql)
	assert.Equal(t, "value", params[0])
}

func TestBuildUpdateSql(t *testing.T) {
	var vars = map[string]string{
		"field": "value",
	}

	sql, params := buildUpdateSql("table_name", "id", 12345, vars)

	assert.Equal(t, "UPDATE `table_name` SET `field`=? WHERE `id` = ?", sql)
	assert.Equal(t, "value", params[0])
	assert.Equal(t, 12345, params[1])
}

func TestPrepareData(t *testing.T) {
	// map[string]string
	assert.Equal(t, map[string]string{"a": "A", "b": "B"}, prepareData(map[string]string{"a": "A", "b": "B"}))

	// map[string]interface
	assert.Equal(t, map[string]string{"a": "A", "n": "12345"}, prepareData(map[string]interface{}{"a": "A", "n": 12345}))

	// map[string]interface containing nested json array
	assert.Equal(t, map[string]string{"json": `["a","b"]`}, prepareData(map[string]interface{}{"json": []string{"a", "b"}}))

	// Struct
	type SomeStruct struct {
		A string  `json:"a"`
		N int     `json:"n"`
		F float32 `json:"f"`
	}
	assert.Equal(t, map[string]string{"a": "A", "n": "12345", "f": "123.456"}, prepareData(SomeStruct{A: "A",
		N: 12345,
		F: 123.456,
	}))

	// key-value
	assert.Equal(t, map[string]string{"a": "A", "n": "12345", "f": "12.345678"}, prepareData("a", "A", "n", 12345, "f", 12.345678))

}
