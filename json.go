package dorm

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// JSON field support
// http://www.booneputney.com/2015-06-18-gorm-golang-jsonb-value-copy/

type JDoc map[string]interface{}

type JArr []interface{}

type JArrStr []string

type JArrInt []int

type JArrFlt []float64

type JRaw []byte

/* JDoc */

func NewJDoc() *JDoc {
	doc := make(JDoc)
	return &doc
}

func NewJDoc2(m map[string]interface{}) *JDoc {
	doc := make(JDoc)
	for key, val := range m {
		doc[key] = val
	}
	return &doc
}

func (j *JDoc) Set(key string, val interface{}) *JDoc {
	(*j)[key] = val
	return j // for chaining
}

func (j *JDoc) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	b, err := json.Marshal(j)
	return string(b), err
}

func (j *JDoc) Scan(value interface{}) error {
	if value == nil {
		j = nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("scan source must be []byte")
	}
	if err := json.Unmarshal(bytes, j); err != nil {
		return err
	}
	return nil
}

/* End JDoc */

/* JArr */

func NewJArr(items ...interface{}) *JArr {
	len := len(items)
	arr := make(JArr, len)
	for i := 0; i < len; i++ {
		arr[i] = items[i]
	}
	return &arr
}

func (j *JArr) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	b, err := json.Marshal(j)
	return string(b), err
}

func (j *JArr) Scan(value interface{}) error {
	if value == nil {
		j = nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("scan source must be []byte")
	}
	if err := json.Unmarshal(bytes, &j); err != nil {
		return err
	}
	return nil
}

func (j *JArr) Contains(v interface{}) bool {
	arr := *j
	for i := range arr {
		if arr[i] == v {
			return true
		}
	}
	return false
}

/* End JArr */

/* JArrStr */

func NewJArrStr(items ...string) *JArrStr {
	len := len(items)
	arr := make(JArrStr, len)
	for i := 0; i < len; i++ {
		arr[i] = items[i]
	}
	return &arr
}

func (j *JArrStr) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	b, err := json.Marshal(j)
	return string(b), err
}

func (j *JArrStr) Scan(value interface{}) error {
	if value == nil {
		j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("scan source must be []byte")
	}
	if err := json.Unmarshal(bytes, &j); err != nil {
		return err
	}
	return nil
}

func (j *JArrStr) Contains(str string) bool {
	arr := *j
	for i := range arr {
		if arr[i] == str {
			return true
		}
	}
	return false
}

/* End JArrStr */

/* JArrInt */

func NewJArrInt(items ...int) *JArrInt {
	len := len(items)
	arr := make(JArrInt, len)
	for i := 0; i < len; i++ {
		arr[i] = items[i]
	}
	return &arr
}

func (j *JArrInt) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	b, err := json.Marshal(j)
	return string(b), err
}

func (j *JArrInt) Scan(value interface{}) error {
	if value == nil {
		j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("scan source must be []byte")
	}
	if err := json.Unmarshal(bytes, &j); err != nil {
		return err
	}
	return nil
}

func (j *JArrInt) Contains(needle int) bool {
	arr := *j
	for i := range arr {
		if arr[i] == needle {
			return true
		}
	}
	return false
}

/* End JArrInt */

/* JArrFlt */

func NewJArrFlt(items ...float64) *JArrFlt {
	len := len(items)
	arr := make(JArrFlt, len)
	for i := 0; i < len; i++ {
		arr[i] = items[i]
	}
	return &arr
}

func (j *JArrFlt) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	b, err := json.Marshal(j)
	return string(b), err
}

func (j *JArrFlt) Scan(value interface{}) error {
	if value == nil {
		j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("scan source must be []byte")
	}
	if err := json.Unmarshal(bytes, &j); err != nil {
		return err
	}
	return nil
}

func (j *JArrFlt) Contains(f float64) bool {
	arr := *j
	for i := range arr {
		if arr[i] == f {
			return true
		}
	}
	return false
}

/* End JArrFlt */

/* JRaw */

func NewJRaw(ifc interface{}) *JRaw {
	bytes, err := json.Marshal(ifc)
	if err != nil {
		panic(err)
	}
	var j JRaw = bytes
	return &j
}

func NewJRaw2(str string) *JRaw {
	var j JRaw = []byte(str)
	return &j
}

func (j *JRaw) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return string(*j), nil
}

func (j *JRaw) Scan(value interface{}) error {
	if value == nil {
		*j = nil // TODO: check?
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("scan source must be []byte")
	}

	*j = bytes
	return nil
}

func (j *JRaw) MarshalJSON() ([]byte, error) {
	return *j, nil
}

func (j *JRaw) UnmarshalJSON(data []byte) error {
	if j == nil {
		return errors.New("JRaw: UnmarshalJSON on nil pointer")
	}
	*j = append((*j)[0:], data...)
	// *j = append((*j)[0:0], data...)
	return nil
}

func (j *JRaw) Obtain() interface{} {
	if j == nil {
		return nil
	}
	var ifc interface{}
	err := json.Unmarshal(*j, ifc)
	if err != nil {
		panic(err)
	}
	return ifc
}

/* End JRaw */
