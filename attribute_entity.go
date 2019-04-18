package dorm

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"bitbucket.org/rightjoin/ion/db/orm"
	"github.com/rightjoin/rutl/conv"
	"github.com/rightjoin/rutl/refl"
)

type AttributeEntity struct {
	PKey

	// What is
	Attribute

	Entity string `sql:"TYPE:varchar(64);not null" json:"entity" insert:"must" update:"no"`
	Field  string `sql:"TYPE:varchar(64);not null;DEFAULT:'info'" json:"field" update:"no" unique:"idx_uniq_key(entity,field,code,active)"`

	// Behaviours
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
var attrMutex sync.Mutex

// TODO:
// attrMap should destroy itself in every 5 min, so that
// any latest changes can go live in 5 min
func loadAttributes() {
	if attrMap != nil {
		return
	}

	attrMutex.Lock()
	{
		dbo := orm.Get(true)
		var attrs []AttributeEntity
		if err := dbo.Find(&attrs).Error; err != nil {
			panic(err)
		}

		attrMap = make(map[string]AttributeEntity)

		for _, a := range attrs {
			codeKey := indexKey(a.Entity, a.Field, a.Code)
			attrMap[codeKey] = a
		}
	}
	attrMutex.Unlock()
}

func indexKey(index ...string) string {
	return strings.Join(index, "___")
}

func AttributeValidate(modl interface{}, data map[string]string) (bool, error) {

	var table = Table(modl)

	// Load all the attributes, if they are not already
	// cached in the global variable
	loadAttributes()

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

		// We need to collage/merge keys of certain types under a
		// single json like fiield, So loop through an aggregate them all
		collated := make(map[string]interface{})
		for key, val := range data {
			if strings.HasPrefix(key, sql+".") {
				// Locate the attribute
				code := strings.Replace(key, sql+".", "", -1)
				attr, found := attrMap[indexKey(table, sql, code)]

				// Barf if not found
				if !found {
					return false, fmt.Errorf("attribute not found: %s", key)
				}

				// Check that the located attribute accepts this
				// type of input value
				item, err := attr.accetps(val)
				if err != nil {
					return false, err
				}

				// all good, so lets collate "property" part of info.property
				collated[code] = item
			}
		}

		// merge all collated items into a single value
		if len(collated) > 0 {
			b, err := json.Marshal(collated)
			if err != nil {
				return false, errors.New("could not encode to json")
			}

			// set the condensed field in the input map
			data[sql] = string(b)

			// unset all the fields that were collated
			for key := range data {
				if strings.HasPrefix(key, sql+".") {
					delete(data, key)
				}
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

	// store in db
	dbo := orm.Get(true)
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
	dbo := orm.Get(true)
	err = UpdateSelect(dbo, "id", id, &att, post)
	if err != nil {
		return nil, err
	}

	return &att, nil
}
