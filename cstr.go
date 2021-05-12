package dorm

import (
	"fmt"
	"net/url"
	"strings"
)

type Connecter interface {
	CStr() string
}

type MysqlConn struct {
	Host         string
	Port         int
	Db           string
	Username     string
	Password     string
	Timezone     string `fig:"optional"`
	ReadTimeout  string `fig:"optional"`
	WriteTimeout string `fig:"optional"`
	Timeout      string `fig:"optional"`
}

// CStr returns the properly formatted connection
// string to connect to mysql database
func (m MysqlConn) CStr() string {

	var params = map[string]string{}

	if m.Timezone != "" {
		params["parseTime"] = "true"
		params["loc"] = url.QueryEscape(m.Timezone)
	}

	if m.ReadTimeout != "" {
		params["readTimeout"] = m.ReadTimeout
	}

	if m.WriteTimeout != "" {
		params["writeTimeout"] = m.WriteTimeout
	}

	if m.Timeout != "" {
		params["timeout"] = m.Timeout
	}

	cstr := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		m.Username, m.Password,
		m.Host, m.Port, m.Db,
	)

	if len(params) > 0 {
		chunks := []string{}
		for key, val := range params {
			chunks = append(chunks, fmt.Sprintf("%s=%s", key, val))
		}
		cstr = cstr + "?" + strings.Join(chunks, "&")
	}

	return cstr
}

type PgConn struct {
	Host     string
	Port     int
	Db       string
	Username string
	Password string
}

func (p PgConn) CStr() string {
	return fmt.Sprintf("host=%s port=%d user=%s dbname=%s password=%s sslmode=disable",
		p.Host, p.Port, p.Username, p.Db, p.Password)
}
