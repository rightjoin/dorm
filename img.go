package dorm

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type Img struct {
	FileID string `json:"file_id"`
	Src    string `json:"src"`
}

type ImgMulti struct {
	Images *[]Img `sql:"TYPE:json" json:"images"`
}

func (p *Img) Value() (driver.Value, error) {
	if p == nil {
		return nil, nil
	}
	str, err := json.Marshal(p)
	return string(str), err
}

func (p *Img) Scan(value interface{}) error {
	if value == nil {
		p = nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("Scan source was not []bytes")
	}
	if err := json.Unmarshal(bytes, &p); err != nil {
		return err
	}
	return nil
}
