package dorm

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

var allow = []string{"int", "decimal", "string", "bool"}

type Attribute struct {
	Code     string `sql:"TYPE:varchar(64);not null" json:"code" insert:"must" update:"no"`
	Name     string `sql:"TYPE:varchar(64);not null" json:"name" insert:"must"`
	Datatype string `sql:"TYPE:enum('int','decimal','string','bool');not null" json:"datatype" insert:"must"`

	Mandatory uint8    `sql:"TYPE:tinyint unsigned;not null;DEFAULT:'0'" json:"mandatory" update:"no"`
	Enums     *JArr    `sql:"TYPE:json" json:"enums"`
	Units     *JArrStr `sql:"TYPE:json;" json:"units"`

	// multi-select flag decides whether attribute of enum type can have multiple values
	MultiSelect *uint8 `sql:"TYPE:tinyint(1) unsigned;not null;DEFAULT:'0'" json:"multi_select" update:"no"`
}

func (a Attribute) Accepts(inp string) (interface{}, error) {

	validate := func(inp string) (interface{}, error) {
		// Check if it can be parsed, and
		// obtain its value
		val, err := a.parse(inp)
		if err != nil {
			return nil, err
		}

		// If enums is defined, then the given value
		// must be one of values defined in enums array.
		// First do a simple value check, and then do a string
		// based check also
		if a.Enums != nil && !a.Enums.Contains(val) {
			found := false
			for i := range *a.Enums {
				item := (*a.Enums)[i]
				if fmt.Sprint(item) == fmt.Sprint(val) {
					found = true
					break
				}
			}

			if !found {
				switch {
				case len(*a.Enums) <= 20:
					return nil, fmt.Errorf("Input %s must be one of enums %v", inp, *a.Enums)
				default:
					return nil, fmt.Errorf("Input %s must be one of enums %v etc", inp, (*a.Enums)[:20])
				}
			}
		}

		// If units are given, then ensure that string datatypes
		// are of format "n unit" or "n.m unit"
		if a.Datatype == "string" && a.Units != nil {
			// this reg ex matches "123.45 m" and "123 m" type inputs
			var sregex = fmt.Sprintf(`^[0-9]+(\.[0-9]{1,})? (%s)$`, strings.Join(*a.Units, "|"))
			var regex = regexp.MustCompile(sregex)
			if !regex.MatchString(inp) {
				return nil, fmt.Errorf("Input %s must be numeric followed by any unit: %v", inp, *a.Units)
			}
		}

		return val, nil
	}

	// Handle multi-select enums
	if a.Enums != nil && a.MultiSelect != nil && *a.MultiSelect == 1 {
		arr := []interface{}{}
		validatedArr := []interface{}{}

		err := json.Unmarshal([]byte(inp), &arr)
		if err != nil {
			return nil, errors.Wrapf(err, "parsing enum multi-select value %s failed", inp)
		}

		// Validate every element in the multi-selected array of enum
		for _, val := range arr {
			v, err := validate(fmt.Sprint(val))
			if err != nil {
				return nil, err
			}

			validatedArr = append(validatedArr, v)
		}

		return validatedArr, nil
	}

	return validate(inp)
}

func (a Attribute) parse(inp string) (interface{}, error) {

	switch a.Datatype {
	case "bool":
		{
			switch inp {
			case "true", "True", "1", "yes", "Yes", "y", "Y":
				return true, nil
			case "false", "False", "0", "no", "No", "n", "N":
				return false, nil
			default:
				return nil, errors.New("can not parse bool:" + inp)
			}
		}
	case "int":
		i, err := strconv.ParseInt(inp, 10, 32)
		return int(i), err
	case "decimal":
		return strconv.ParseFloat(inp, 64)
	case "string":
		return inp, nil
	}

	return nil, errors.New("unknown attribute datatype")
}
