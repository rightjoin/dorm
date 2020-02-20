package dorm

import (
	"fmt"
	"net/url"
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
	ConnTimeout  string `fig:"optional"`
}

// CStr returns the properly formatted connection
// string to connect to mysql database
func (m MysqlConn) CStr() string {

	if m.Timezone == "" {
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
			m.Username, m.Password,
			m.Host, m.Port, m.Db,
		)
	}

	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&loc=%s",
		m.Username, m.Password,
		m.Host, m.Port, m.Db,
		url.QueryEscape(m.Timezone),
	)
}
