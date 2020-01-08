package dorm

import (
	"fmt"
	"math/rand"
	"net/url"
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/rightjoin/fig"
)

// list of connections
var connections = make(map[string]*gorm.DB)

var slaveKeys []string

// list of intialization functions that need to be
// called when a new db-connection is established.
// initlized with a value to use singular table names
var initializations = []func(*gorm.DB){
	func(dbo *gorm.DB) {
		dbo.SingularTable(true)
	},
}

func OnInitialize(fn func(*gorm.DB)) {
	// http://www.alexedwards.net/blog/configuring-sqldb
	// http://techblog.en.klab-blogs.com/archives/31093990.html
	initializations = append(initializations, fn)
}

func GetORM(master bool) *gorm.DB {
	if master {
		return GetORMConfig("database.master")
	}

	if slaveKeys == nil {
		slaveKeys = make([]string, 0)
		if fig.Exists("database.slaves") {
			slaves := fig.Map("database.slaves")
			for key := range slaves {
				slaveKeys = append(slaveKeys, key)
			}
		}
	}

	if len(slaveKeys) == 0 {
		return GetORM(true)
	}

	randomSlave := slaveKeys[rand.Intn(len(slaveKeys))]
	return GetORMConfig("database.slaves", randomSlave)
}

func GetORMConfig(container ...string) *gorm.DB {

	parent := strings.Join(container, ".")
	fig.MustExist(parent)

	engine := fig.String(parent, "engine")
	cstr := GetCstrConfig(engine, parent)
	return GetORMCstr(engine, cstr)
}

func GetORMCstr(engine string, conn string) *gorm.DB {

	key := engine + ":" + conn
	var orm *gorm.DB
	var ok bool
	var e error

	if orm, ok = connections[key]; ok {
		return orm.Unscoped()
	}

	// http://go-database-sql.org/accessing.html
	// the sql.DB object is designed to be long-lived
	if orm, e = gorm.Open(engine, conn); e == nil {

		// run the initializations on this object
		if initializations != nil {
			for _, fn := range initializations {
				fn(orm)
			}
		}

		// store this object
		connections[key] = orm

		return orm.Unscoped()
	}
	panic(e)
}

// GetCstrConfig reads the configuratin and returns the Cstr
func GetCstrConfig(engine string, container ...string) string {
	parent := strings.Join(container, ".")

	switch engine {
	case "mysql":
		my := MysqlConn{}
		fig.Struct(&my, parent)
		return GetCstr(engine, map[string]interface{}{
			"username":     my.Username,
			"password":     my.Password,
			"host":         my.Host,
			"port":         my.Port,
			"db":           my.Db,
			"timezone":     my.Timezone,
			"readTimeout":  my.ReadTimeout,
			"writeTimeout": my.WriteTimeout,
			"timeout":      my.ConnTimeout,
		})

	default:
		panic("unsupported db engine:" + engine)
	}
}

func GetCstr(engine string, prop map[string]interface{}) (cstr string) {

	switch engine {
	case "mysql":
		if timezone, ok := prop["timezone"]; !ok || timezone == "" {
			prop["timezone"] = "UTC"
		}
		cstr = fmt.Sprintf("%s:%s@tcp(%s:%v)/%s?parseTime=true&loc=%s",
			prop["username"], prop["password"],
			prop["host"], prop["port"], prop["db"],
			url.QueryEscape(prop["timezone"].(string)),
		)

		if readTimeout, ok := prop["readTimeout"]; ok && readTimeout != "" {
			if !strings.ContainsAny(fmt.Sprint(readTimeout), "smh") {
				panic("timeout must be mentioned alongwith m,s,ms or h time's unit suffix")
			}
			cstr += fmt.Sprintf("&readTimeout=%s", readTimeout)
		}
		if writeTimeout, ok := prop["writeTimeout"]; ok && writeTimeout != "" {
			if !strings.ContainsAny(fmt.Sprint(writeTimeout), "smh") {
				panic("timeout must be mentioned alongwith m,s,ms or h time's unit suffix")
			}
			cstr += fmt.Sprintf("&writeTimeout=%s", writeTimeout)
		}
		if timeout, ok := prop["timeout"]; ok && timeout != "" {
			if !strings.ContainsAny(fmt.Sprint(timeout), "smh") {
				panic("timeout must be mentioned alongwith m,s,ms or h time's unit suffix")
			}
			cstr += fmt.Sprintf("&timeout=%s", timeout)
		}
	default:
		panic("blah blah!")
	}

	return
}
