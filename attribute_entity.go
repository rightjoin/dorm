package dorm

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/rightjoin/rutl/conv"
	"github.com/rightjoin/rutl/refl"
)

type AttributeEntity struct {
	PKey

	// What is
	Attribute

	Entity string `sql:"TYPE:varchar(64);not null" json:"entity" insert:"must" update:"no"`
	Field  string `sql:"TYPE:varchar(64);not null;DEFAULT:'info'" json:"field" update:"no" unique:"idx_uniq_key(entity,field,code)"`

	// Behaviours
	// ToDo: Add SoftDelete
	Active1
	Historic
	WhosThat
	Timed
}

func (a AttributeEntity) Triggers() []string {
	return []string{
		`CREATE TRIGGER attribute_entity_bfr_insert BEFORE INSERT ON attribute_entity FOR EACH ROW
        BEGIN
			SET NEW.code = REPLACE(LCASE(TRIM(NEW.code)),' ','-'); #no spaces, lowercase
			SET NEW.code = REPLACE(NEW.code,'.',''); #no dots
        END`,
		`CREATE TRIGGER attribute_entity_bfr_update BEFORE UPDATE ON attribute_entity FOR EACH ROW
        BEGIN
			SET NEW.code = REPLACE(LCASE(TRIM(NEW.code)),' ','-'); #no spaces, lowercase
			SET NEW.code = REPLACE(NEW.code,'.',''); #no dots
		END`,
	}
}

var attrMap map[string]AttributeEntity
var mandatoryAttr map[string]bool
var attrMutex sync.Mutex

// TODO:
// attrMap should destroy itself in every 5 min, so that
// any latest changes can go live in 5 min
func loadAttributes() {
	attrMutex.Lock()
	{
		dbo := GetORM(true)
		var attrs []AttributeEntity
		if err := dbo.Where("active = 1").Find(&attrs).Error; err != nil {
			panic(err)
		}

		attrMap = make(map[string]AttributeEntity)
		mandatoryAttr = make(map[string]bool)

		for _, a := range attrs {
			codeKey := indexKey(a.Entity, a.Field, a.Code)
			attrMap[codeKey] = a

			// Set mandatory flag against the attribute
			if a.Mandatory > 0 {
				mandatoryAttr[indexKey(a.Entity, a.Code)] = true
			}
		}
	}
	attrMutex.Unlock()
}

func indexKey(index ...string) string {
	return strings.Join(index, "___")
}

func AttributeValidate(modl interface{}, data map[string]string, action string) (bool, error) {

	if action != "insert" && action != "update" {
		return false, errors.New("unknown action : " + action)
	}

	var table = Table(modl)

	// Load all the attributes, if they are not already
	// cached in the global variable
	loadAttributes()

	// Locates an attribute and returns it's valid-value(value that's accepted by the attr)
	validateReturnItem := func(code, val, sql string) (interface{}, error) {

		// locate the attribute(i.e. code)
		attr, found := attrMap[indexKey(table, sql, code)]
		if !found {
			return nil, fmt.Errorf("attribute not found: %s", code)
		}

		// Check that the located attribute accepts this
		// type of input value
		item, err := attr.Accepts(val)
		if err != nil {
			return false, err
		}

		return item, nil
	}

	// We need to collage/merge keys of certain types under a
	// single json like field, So loop through an aggregate them all
	collated := make(map[string]interface{})

	// Iterate over each of the fields of the struct represented by model,
	// and check if needs validation
	for _, fld := range refl.NestedFields(modl) {
		//sgnt := refl.Signature(fld.Type)
		sql := conv.CaseSnake(fld.Name)

		// Ignore certain kinds of fields, as they don't require
		// any validations
		if sql == "who" {
			continue
		}

		// TODO:
		// for what field types should this be done? info/map/what else?

		// Handles cases where the input is already a json
		if sql == "info" {
			if info, ok := data["info"]; ok {
				infoMap := make(map[string]interface{})
				if err := json.Unmarshal([]byte(info), &infoMap); err != nil {
					return false, err
				}

				for key, val := range infoMap {

					item, err := validateReturnItem(key, fmt.Sprint(val), sql)
					if err != nil {
						return false, err
					}

					collated[key] = item
				}
			}
		}
	}

	prefix := "info."

	// collates form data or data in the format of "someObj.someKey"
	for key, val := range data {

		// Handles all the cases where the input is in the form of "info.key",
		// i.e. if it's a form data, would fail incase of json/application
		if strings.HasPrefix(key, prefix) {

			code := strings.Replace(key, prefix, "", -1)

			item, err := validateReturnItem(code, val, "info")
			if err != nil {
				return false, err
			}

			// all good, so lets collate "property" part of info.property
			collated[code] = item
		}

	}

	// Check for availability of mandatory fields
	// Note: for updates, we are ignoring the mandatory check
	if len(collated) == 0 && len(mandatoryAttr) > 0 && action != "update" {
		for key := range mandatoryAttr {
			if table == strings.Split(key, "___")[0] {
				return false, fmt.Errorf("mandatory attribute_entity missing %s", strings.Split(key, "___")[1])
			}
		}
	}

	// merge all collated items into a single value
	if len(collated) > 0 {

		// Validate the presence of mandatory attr
		// only incase of an insert operation
		if action == "insert" {
			for key := range mandatoryAttr {

				// need to check for mandatory attributes of certain kind
				// i.e. article or article_parent
				entity := strings.Split(key, "___")[0]
				code := strings.Split(key, "___")[1]
				if entity != table {
					continue
				}

				if _, ok := collated[code]; !ok {
					return false, fmt.Errorf("mandatory attribute_entity %s missing", code)
				}

			}
		}

		b, err := json.Marshal(collated)
		if err != nil {
			return false, errors.New("could not encode to json")
		}

		// set the condensed field in the input map
		data["info"] = string(b)

		// unset all the fields that were collated
		for key := range data {
			if strings.HasPrefix(key, prefix) {
				delete(data, key)
			}
		}
	}

	// all good
	return true, nil
}

func AttributeInsertViaEntity(post map[string]string, entity string, field string) (*AttributeEntity, error) {
	var att AttributeEntity
	var err error

	post["entity"] = entity
	post["field"] = field

	if units, ok := post["units"]; ok {
		parsedArr := []string{}
		err := json.Unmarshal([]byte(units), &parsedArr)
		if err == nil {
			if len(parsedArr) == 0 {
				delete(post, "units")
			}
		}
	}

	if enums, ok := post["enums"]; ok {
		parsedArr := []string{}
		err := json.Unmarshal([]byte(enums), &parsedArr)
		if err == nil {
			if len(parsedArr) == 0 {
				delete(post, "enums")
			}
		}
	}

	// store in db
	dbo := GetORM(true)
	err = InsertSelect(dbo, &att, post)
	if err != nil {
		return nil, err
	}

	return &att, nil
}

func AttributeUpdateViaEntity(post map[string]string, id string) (*AttributeEntity, error) {
	var att AttributeEntity
	var err error

	// store in db
	dbo := GetORM(true)
	err = UpdateSelect(dbo, "id", id, &att, post)
	if err != nil {
		return nil, err
	}

	return &att, nil
}
