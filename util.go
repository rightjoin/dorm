package dorm

import (
	"reflect"

	"github.com/rightjoin/utila/conv"
)

const (
	NullString = "^NULL^"
)

func tableName(model interface{}) string {
	t := reflect.TypeOf(model)
	v := reflect.ValueOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		v = v.Elem()
	}

	if _, ok := t.MethodByName("TableName"); ok {
		name := v.MethodByName("TableName").Call([]reflect.Value{})
		return name[0].String()
	}
	return conv.CaseSnake(t.Name())
}
