package dorm

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/rightjoin/rutl/refl"
)

// MySQL doesn't support nested transactions. So the code tries to start
// a transaction. If there is an error, it assumes that a new transaction
// cannot be initiated, and hence a transaction must already be running

func InsertSelect(dbo *gorm.DB, addr interface{}, data ...interface{}) error {

	// prepare data from user input
	input := prepareData(data...)

	// do validations of model fields
	if ok, errs := validateModel(addr, input, "insert"); !ok {
		return errs[0]
	}

	txn := dbo.Begin()
	if txn.Error != nil {
		// transaction must already be running,
		// so we just send the insert to db
		return doInsertion(dbo, addr, input, true)
	}

	// execute insertion
	err := doInsertion(txn, addr, input, true)
	if err != nil {
		txn.Rollback()
		return err
	}

	return txn.Commit().Error
}

func Insert(dbo *gorm.DB, addr interface{}, data ...interface{}) error {

	// prepare data from user input
	input := prepareData(data...)

	// do validations of model fields
	if ok, errs := validateModel(addr, input, "insert"); !ok {
		return errs[0]
	}

	txn := dbo.Begin()
	if txn.Error != nil {
		// transaction must already be running,
		// so we just send the insert to db
		return doInsertion(dbo, addr, input, false)
	}

	// execute insertion
	err := doInsertion(txn, addr, input, false)
	if err != nil {
		txn.Rollback()
		return err
	}

	return txn.Commit().Error
}

func UpdateSelect(dbo *gorm.DB, pkField string, pkValue interface{}, addr interface{}, data ...interface{}) error {

	// prepare data from user input
	input := prepareData(data...)

	// do validations of model fields
	if ok, errs := validateModel(addr, input, "update"); !ok {
		return errs[0]
	}
	txn := dbo.Begin()
	if txn.Error != nil {
		// transaction must already be running,
		// so we just send the update to db
		return doUpdation(dbo, pkField, pkValue, addr, input, true)
	}

	// execute updation
	err := doUpdation(txn, pkField, pkValue, addr, input, true)
	if err != nil {
		txn.Rollback()
		return err
	}

	return txn.Commit().Error
}

func Update(dbo *gorm.DB, pkField string, pkValue interface{}, addr interface{}, data ...interface{}) error {

	// prepare data from user input
	input := prepareData(data...)

	// do validations of model fields
	if ok, errs := validateModel(addr, input, "update"); !ok {
		return errs[0]
	}

	txn := dbo.Begin()
	if txn.Error != nil {
		// transaction must already be running,
		// so we just send the update to db
		return doUpdation(dbo, pkField, pkValue, addr, input, false)
	}

	// execute updation
	err := doUpdation(txn, pkField, pkValue, addr, input, false)
	if err != nil {
		txn.Rollback()
		return err
	}

	return txn.Commit().Error
}

// prepareData takes a set of inputs and converts it into a map[string]string.
// If there is only one input and it happens to be a map[string]string it
// is used as return value. If this input happens to be a map[string]interface{}
// then it gets converted into map[string]string. If it happens to be a struct
// then it is converted to map[string]interface, and finally to map[string]string
// If neither then the input is treated as a series of key-value pairs.
func prepareData(data ...interface{}) map[string]string {

	var inp map[string]interface{}

	// if length is 1, then the given input must be a map
	if len(data) == 1 {

		// if it is a map of string -> string, we are all good
		if values, ok := data[0].(map[string]string); ok {
			return values
		}

		// if it is a map of string -> interface then we will process it further
		if temp, ok := data[0].(map[string]interface{}); ok {
			inp = temp
		} else if reflect.TypeOf(data[0]).Kind() == reflect.Struct {
			b, err := json.Marshal(data[0])
			if err != nil {
				panic("can not convert struct to map")
			}
			err = json.Unmarshal(b, &inp)
			if err != nil {
				panic("can not retrieve map from struct")
			}
		} else {
			panic("unhandled type (" + reflect.TypeOf(data[0]).Name() + ") passed to prepareData")
		}

	} else {

		// must be even number inputs
		if len(data)%2 != 0 {
			panic("key-value pairs must be have even inputs")
		}

		// load data into "inp" map
		inp = make(map[string]interface{})
		for i := 0; i < len(data); i = i + 2 {
			inp[fmt.Sprint(data[i])] = data[i+1]
		}
	}

	// convert values to into appropriate string types
	// because insert/update statements need string values
	var mStr = make(map[string]string)
	for k, v := range inp {
		switch val := v.(type) {
		case int, int16, int64, int8, uint, uint8, uint16, uint32, uint64:
			mStr[k] = fmt.Sprintf("%d", val)
		case string:
			mStr[k] = val
		case float32:
			mStr[k] = strconv.FormatFloat(float64(val), 'f', -1, 64)
		case float64:
			mStr[k] = strconv.FormatFloat(val, 'f', -1, 64)
		case time.Time:
			mStr[k] = val.Format("2006-01-02T15:04:05.999999 -0700")
			//mStr[k] = val.Format(time.RFC3339)
		case bool:
			if val {
				mStr[k] = "1"
			}
			mStr[k] = "0"
		default:
			// If it is a slice, then try to encode it in a string
			kind := reflect.TypeOf(v).Kind()
			if kind == reflect.Slice || kind == reflect.Array {
				b, err := json.Marshal(v)
				if err != nil {
					panic("could not convert array/slice to json format")
				}
				mStr[k] = string(b)
			} else {
				mStr[k] = fmt.Sprint(v)
			}
		}
	}

	return mStr
}

func doInsertion(txn *gorm.DB, addr interface{}, data map[string]string, doRead bool) error {

	// get table name
	table := Table(addr)

	// ensure that "who" values have been attached from request
	if refl.ComposedOf(addr, WhosThat{}) {
		if _, ok := data["who"]; !ok {
			return fmt.Errorf("insert data missing 'who' content")
		}
	}

	// send insert to db
	sql, params := buildInsertSql(table, data)
	err := txn.Exec(sql, params...).Error
	if err != nil {
		return err
	}

	// does model have PreCommit validation
	v := reflect.ValueOf(addr).Elem()
	_, found := v.Interface().(hookCommit)
	if found {
		doRead = true // force-read to perform PreCommit validations
	}

	if doRead {
		// fetch primary key value
		var pid int
		row := txn.Raw("SELECT LAST_INSERT_ID()").Row()
		err = row.Scan(&pid)
		if err != nil {
			return err
		}

		// select record
		//err = txn.Where("id=?", pid).Find(addr).Error
		err = txn.Raw("SELECT * FROM "+Table(addr)+" WHERE id=?", pid).Scan(addr).Error
		if err != nil {
			return err
		}

		// invoke PreCommit validations
		if found {
			out := v.MethodByName("PreCommit").Call([]reflect.Value{})
			if !out[0].IsNil() {
				return out[0].Interface().(error)
			}
		}
	}

	return nil
}

func doUpdation(txn *gorm.DB, pkField string, pkValue interface{}, addr interface{}, data map[string]string, doRead bool) error {

	// get table name
	table := Table(addr)

	// ensure that "who" values have been attached from request
	if refl.ComposedOf(addr, WhosThat{}) {
		if _, ok := data["who"]; !ok {
			return fmt.Errorf("update data missing 'who' content")
		}
	}

	// send update to db
	sql, params := buildUpdateSql(table, pkField, pkValue, data)
	fmt.Println("sql:", sql)
	err := txn.Exec(sql, params...).Error
	if err != nil {
		return err
	}

	// does model have PreCommit validation
	v := reflect.ValueOf(addr).Elem()
	_, found := v.Interface().(hookCommit)
	if found {
		doRead = true // force-read to perform PreCommit validations
	}

	if doRead {
		// select record
		err = txn.Where(pkField+"=?", pkValue).Find(addr).Error
		if err != nil {
			return err
		}

		// invoke PreCommit validations
		if found {
			out := v.MethodByName("PreCommit").Call([]reflect.Value{})
			if !out[0].IsNil() {
				return out[0].Interface().(error)
			}
		}
	}

	return nil
}

func buildInsertSql(tbl string, inp map[string]string) (string, []interface{}) {

	// TODO: optimize string concatenation

	params := make([]interface{}, 0)

	keys := ""
	vals := ""
	for key, val := range inp {
		if val == NullString { // null check
			if keys == "" {
				keys = fmt.Sprintf("`%s`", key)
				vals = "NULL"
			} else {
				keys += fmt.Sprintf(", `%s`", key)
				vals += ", NULL"
			}
		} else {
			if keys == "" {
				keys = fmt.Sprintf("`%s`", key)
				vals = "?"
			} else {
				keys += fmt.Sprintf(", `%s`", key)
				vals += ", ?"
			}
			if EncryptColumn != nil {
				val = EncryptColumn(tbl, key, val)
			}
			params = append(params, val)
		}
	}
	return fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s)", tbl, keys, vals), params
}

func buildUpdateSql(tbl string, pkField string, pkValue interface{}, inp map[string]string) (string, []interface{}) {

	params := make([]interface{}, 0)

	upd := ""
	for key, val := range inp {
		if val == NullString { // null check
			if upd == "" {
				upd = fmt.Sprintf("`%s`=NULL", key)
			} else {
				upd += fmt.Sprintf(", `%s`=NULL", key)
			}
		} else {
			if upd == "" {
				upd = fmt.Sprintf("`%s`=?", key)
			} else {
				upd += fmt.Sprintf(", `%s`=?", key)
			}
			if EncryptColumn != nil {
				val = EncryptColumn(tbl, key, val)
			}
			params = append(params, val)
		}
	}

	// add search criterion at the end
	params = append(params, pkValue)

	return fmt.Sprintf("UPDATE `%s` SET %s WHERE `%s` = ?", tbl, upd, pkField), params
}

var EncryptColumn func(tbl string, field string, value string) (encrpValue string)

type hookCommit interface {
	PreCommit() error
}
