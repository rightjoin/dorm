package dorm

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	refl2 "bitbucket.org/rightjoin/ion/refl"
	"github.com/asaskevich/govalidator"
	"github.com/rightjoin/utila/refl"
	"github.com/rightjoin/utila/txt"
)

func validateModel(modl interface{}, data map[string]string, action string) (bool, []error) {

	var errs = make([]error, 0)

	obj := modl
	mname := ""

	// input validation
	if action != "insert" && action != "update" {
		panic("action should be insert|update")
	}

	// resolve indirection (if present).
	// also extract model name.
	{
		rval := reflect.ValueOf(modl)
		if rval.Kind() == reflect.Ptr {
			rval = rval.Elem()
			obj = rval.Interface()
		}
		mname = rval.Type().Name()
	}

	for _, fld := range refl.NestedFields(obj) {
		fname := fld.Name
		sqlName := txt.CaseSnake(fname)
		sig := signature(fld.Type)
		sig2 := refl2.TypeSignature(fld.Type)
		_, hasData := data[sqlName]

		fmt.Println("SQLName:", sqlName, "SIGNATURE:", sig, sig2, "FOUND:", hasData, "FLDTYPE:", fld.Type)

		// auto trim string fields unless "trim" set to "no"
		if hasData && (sig == "string" || sig == "*string") && fld.Tag.Get("trim") != "no" {
			data[sqlName] = strings.TrimSpace(data[sqlName])
		}

		// must fields should be present
		if hasData == false && fld.Tag.Get(action) == "must" {
			errs = append(errs, fmt.Errorf("compulsory field missing during %s: %s.%s", action, mname, sqlName))
		}

		// unwanted fields should not be present
		if hasData == true && fld.Tag.Get(action) == "no" {
			errs = append(errs, fmt.Errorf("forbidden field found during %s: %s.%s", action, mname, sqlName))
		}

		// json validations : json_array, json_map
		if hasData && data[sqlName] != NullString {
			switch sig {
			case "sl:.", "*sl:.":
				{
					var test []interface{}
					if err := json.Unmarshal([]byte(data[sqlName]), &test); err != nil {
						errs = append(errs, fmt.Errorf("json array expected during %s: %s.%s", action, mname, sqlName))
					}
				}
			case "map", "*map":
				{
					var test map[string]interface{}
					if err := json.Unmarshal([]byte(data[sqlName]), &test); err != nil {
						errs = append(errs, fmt.Errorf("json document expected during %s: %s.%s", action, mname, sqlName))
					}
				}

			case "sl:int", "*sl:int":
				{
					var test []int
					if err := json.Unmarshal([]byte(data[sqlName]), &test); err != nil {
						errs = append(errs, fmt.Errorf("json array of int expected during %s: %s.%s", action, mname, sqlName))
					}
				}

			case "sl:string", "*sl:string":
				{
					var test []string
					if err := json.Unmarshal([]byte(data[sqlName]), &test); err != nil {
						errs = append(errs, fmt.Errorf("json array of string expected during %s: %s.%s", action, mname, sqlName))
					}
				}
			default:
				panic("stopping")
			}
		}

		// execute special validations
		if hasData && (sig == "string" || sig == "*string") && fld.Tag.Get("validate") != "" {
			switch fld.Tag.Get("validate") {
			case "email":
				if !govalidator.IsEmail(data[sqlName]) {
					errs = append(errs, fmt.Errorf("field validation (%s) failed found during %s: %s.%s", fld.Tag.Get("validate"), action, mname, sqlName))
				}
				// more validations go here
			}
		}
	}

	if len(errs) == 0 {
		return true, nil
	}

	return false, errs
}
