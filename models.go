package dorm

import (
	"reflect"
	"time"

	"bitbucket.org/rightjoin/ion/db"
	"github.com/rightjoin/utila/txt"
)

type PK struct {
	ID uint `sql:"auto_increment;not null;primary_key" json:"id" insert:"no" update:"no"`
}

type UID10 struct {
	UID string `sql:"TYPE:varchar(10);not null;DEFAULT:'';" json:"uid" unique:"true" insert:"no" update:"no"`
}

type Timed struct {
	CreatedAt time.Time `sql:"TYPE:datetime;not null;DEFAULT:current_timestamp" json:"created_at" insert:"no" update:"no"`
	UpdatedAt time.Time `sql:"TYPE:datetime;not null;DEFAULT:current_timestamp" json:"updated_at" insert:"no" update:"no" index:"true"`
}

type Historic struct {
}

type Seo struct {
	URL          string   `sql:"TYPE:varchar(256);not null;DEFAULT:''" json:"url" unique:"true"`
	URLPast      *db.JArr `sql:"TYPE:json;" json:"-" insert:"no" update:"no"`
	MetaTitle    string   `sql:"TYPE:varchar(256);not null;DEFAULT:''" json:"meta_title"`
	MetaDesc     string   `sql:"TYPE:varchar(256);not null;DEFAULT:''" json:"meta_desc"`
	MetaKeywords string   `sql:"TYPE:varchar(256);not null;DEFAULT:''" json:"meta_keywords"`
	Sitemap      uint8    `sql:"TYPE:tinyint(1) unsigned;not null;DEFAULT:'1'" json:"sitemap"`
}

// UrlColumn gives you the column/field to be used to supply text
// that will form the URL. Typically (and default) is to use a field named
// "name" for population of url_web
func (s Seo) UrlColumn(addr interface{}) string {
	t := reflect.TypeOf(addr)
	t = t.Elem()
	sf, found := t.FieldByName("Seo")
	if !found {
		panic("Seo field not found in model")
	}

	if col, ok := sf.Tag.Lookup("url_column"); ok {
		return col
	}
	return "name"
}

// UrlPrefix gives you the prefix that should be used in the URLs.
// Default is to use "table-name"
func (s Seo) UrlPrefix(addr interface{}) string {
	t := reflect.TypeOf(addr)
	t = t.Elem()
	sf, found := t.FieldByName("Seo")
	if !found {
		panic("Seo field not found in model")
	}

	prefix := tableName(addr)
	if pre, ok := sf.Tag.Lookup("url_prefix"); ok {
		prefix = pre
	}

	return txt.CaseURL(prefix)
}
