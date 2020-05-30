package dorm

import (
	"reflect"
	"strings"
	"time"

	"github.com/rightjoin/rutl/conv"
)

type PKey struct {
	ID uint `sql:"auto_increment;not null;primary_key" json:"id" insert:"no" update:"no"`
}

type UID10 struct {
	UID string `sql:"TYPE:varchar(10) binary;not null;DEFAULT:'';" json:"uid" unique:"true" insert:"no" update:"no"`
}

type UID8 struct {
	UID string `sql:"TYPE:varchar(8) binary;not null;DEFAULT:'';" json:"uid" unique:"true" insert:"no" update:"no"`
}

type UID16 struct {
	UID string `sql:"TYPE:varchar(16) binary;not null;DEFAULT:'';" json:"uid" unique:"true" insert:"no" update:"no"`
}

type Timed struct {
	CreatedAt time.Time `sql:"TYPE:datetime;not null;DEFAULT:current_timestamp" json:"created_at" insert:"no" update:"no"`
	UpdatedAt time.Time `sql:"TYPE:datetime;not null;DEFAULT:current_timestamp" json:"updated_at" insert:"no" update:"no" index:"true"`
}

type TimedLite struct {
	CreatedAt time.Time `sql:"TYPE:datetime;not null;DEFAULT:current_timestamp" json:"created_at" insert:"no" update:"no"`
	UpdatedAt time.Time `sql:"TYPE:datetime;not null;DEFAULT:current_timestamp" json:"updated_at" insert:"no" update:"no"`
}

type Timed4 struct {
	CreatedAt time.Time `sql:"TYPE:datetime(4);not null;DEFAULT:current_timestamp(4)" json:"created_at" insert:"no" update:"no"`
	UpdatedAt time.Time `sql:"TYPE:datetime(4);not null;DEFAULT:current_timestamp(4)" json:"updated_at" insert:"no" update:"no" index:"true"`
}

type Timed4Lite struct {
	CreatedAt time.Time `sql:"TYPE:datetime(4);not null;DEFAULT:current_timestamp(4)" json:"created_at" insert:"no" update:"no"`
	UpdatedAt time.Time `sql:"TYPE:datetime(4);not null;DEFAULT:current_timestamp(4)" json:"updated_at" insert:"no" update:"no"`
}

type Historic struct {
}

type WhosThat struct {
	Who *JDoc `sql:"TYPE:json" json:"-"`
}

type Active1 struct {
	Active uint8 `sql:"TYPE:tinyint(1) unsigned;not null;DEFAULT:'1'" json:"active"`
}

type Active0 struct {
	Active uint8 `sql:"TYPE:tinyint(1) unsigned;not null;DEFAULT:'0'" json:"active"`
}

type Tagged struct {
	Tags *JArrStr `sql:"TYPE:json;" json:"tags"`
}

type Ordered struct {
	Sequence uint `sql:"not null;DEFAULT:'1'" json:"sequence"`
}

type Boosted struct {
	Boost int8 `sql:"not null;DEFAULT:'0'" json:"boost"`
}

type SoftDelete struct {
	Deleted   uint8      `sql:"TYPE:tinyint(1) unsigned;not null;DEFAULT:'0'" json:"deleted" insert:"no" index:"true"`
	DeletedAt *time.Time `sql:"TYPE:datetime;null;" json:"deleted_at" insert:"no" update:"no"`
}

type SoftDelete4 struct {
	Deleted   uint8      `sql:"TYPE:tinyint(1) unsigned;not null;DEFAULT:'0'" json:"deleted" insert:"no" index:"true"`
	DeletedAt *time.Time `sql:"TYPE:datetime(4);null;" json:"deleted_at" insert:"no" update:"no"`
}

type DynamicField struct {
	Info *JDoc `sql:"TYPE:json" json:"info"`
}

type MyISAM struct {
}

type SeoField struct {
	Seo *JDoc  `sql:"TYPE:json" json:"seo"`
	URL string `sql:"TYPE:varchar(256);not null;DEFAULT:''" json:"url" unique:"true"`
}

// type Seo struct {
// 	URL          string `sql:"TYPE:varchar(256);not null;DEFAULT:''" json:"url" unique:"true"`
// 	URLPast      *JArr  `sql:"TYPE:json;" json:"-" insert:"no" update:"no"`
// 	MetaTitle    string `sql:"TYPE:varchar(256);not null;DEFAULT:''" json:"meta_title"`
// 	MetaDesc     string `sql:"TYPE:varchar(256);not null;DEFAULT:''" json:"meta_desc"`
// 	MetaKeywords string `sql:"TYPE:varchar(256);not null;DEFAULT:''" json:"meta_keywords"`
// 	Sitemap      uint8  `sql:"TYPE:tinyint(1) unsigned;not null;DEFAULT:'1'" json:"sitemap"`
// }

// UrlColumn gives you the column/field to be used to supply text
// that will form the URL. Typically (and default) is to use a field named
// "name" for population of url_web
func (s SeoField) UrlColumn(addr interface{}) string {
	t := reflect.TypeOf(addr)
	t = t.Elem()
	sf, found := t.FieldByName("SeoField")
	if !found {
		panic("SeoField field not found in model")
	}

	if col, ok := sf.Tag.Lookup("url_column"); ok {
		return col
	}
	return "name"
}

// GetURLRef will return the fields that will be required in cases where
// the url_column is not directly present in the current model.
func (s SeoField) GetURLRef(addr interface{}) (string, string, string) {
	t := reflect.TypeOf(addr)
	t = t.Elem()
	sf, found := t.FieldByName("SeoField")
	if !found {
		panic("SeoField field not found in model")
	}

	refs := sf.Tag.Get("url_column_ref")
	if refs == "" {
		return "DUAL", "true", "*"
	}

	cols := strings.Split(refs, ",")
	if len(cols) < 3 {
		panic("provide three set of value incase of url_column_ref")
	}

	// Model-to-query, col-to-query-upon, column-to-fetch
	return cols[0], cols[1], cols[2]
}

// UrlPrefix gives you the prefix that should be used in the URLs.
// Default is to use "table-name"
func (s SeoField) UrlPrefix(addr interface{}) string {
	t := reflect.TypeOf(addr)
	t = t.Elem()
	sf, found := t.FieldByName("SeoField")
	if !found {
		panic("Seo field not found in model")
	}

	prefix := Table(addr)
	if pre, ok := sf.Tag.Lookup("url_prefix"); ok {
		prefix = pre
	}

	return conv.CaseURL(prefix)
}
