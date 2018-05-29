package dorm

import (
	"fmt"
	"reflect"

	"github.com/rightjoin/utila/txt"
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
	return txt.CaseSnake(t.Name())
}

func signature(t reflect.Type) string {
	sig := ""
	if t.Kind() == reflect.Ptr {
		sig = "*" + signature(t.Elem())
	} else if t.Kind() == reflect.Map {
		sig = "map"
	} else if t.Kind() == reflect.Struct {
		sig = fmt.Sprintf("st:%s.%s", t.PkgPath(), t.Name())
	} else if t.Kind() == reflect.Interface {
		sig = fmt.Sprintf("i:%s.%s", t.PkgPath(), t.Name())
	} else if t.Kind() == reflect.Array {
		sig = fmt.Sprintf("sl:%s.%s", t.Elem().PkgPath(), t.Elem().Name())
	} else if t.Kind() == reflect.Slice {
		sig = fmt.Sprintf("sl:%s.%s", t.Elem().PkgPath(), t.Elem().Name())
	} else {
		sig = t.Name()
	}
	return sig
}
