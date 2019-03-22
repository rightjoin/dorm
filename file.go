package dorm

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"net/http"

	log "github.com/rightjoin/rlog"
	"github.com/rightjoin/rutl/conv"

	"github.com/rightjoin/rutl/refl"
)

var fileSign = "*st:github.com/rightjoin/dorm.File"

// FileRef global variable controls which of the two references
// (UID or ID) of File is stored as Ref in File. This is used
// by the Media.File() method
var FileRef = "UID" // Possible values are UID or ID

type File struct {
	Ref string `json:"ref"`
	Src string `json:"src"`
}

type ImageFiles struct {
	Images *FileList `sql:"TYPE:json" json:"images"`
}

type FileList []File

func (p *File) Value() (driver.Value, error) {
	if p == nil {
		return nil, nil
	}
	str, err := json.Marshal(p)
	return string(str), err
}

func (p *File) Scan(value interface{}) error {
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

func (p *FileList) Value() (driver.Value, error) {
	if p == nil {
		return nil, nil
	}
	str, err := json.Marshal(p)
	return string(str), err
}

func (p *FileList) Scan(value interface{}) error {
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

// SaveAnyFile first traverses all the fields of a given model. If it
// finds any dorm.Img field, then it tries to see if a file is part of
// post data. If it finds any such files, then it saves them to disk
// and updates the post data, so that appropriate references be stored
// in the corresponding entity table
func SaveAnyFile(req *http.Request, post map[string]string, model interface{}) error {

	for _, fld := range refl.NestedFields(model) {
		if refl.Signature(fld.Type) != fileSign {
			continue
		}

		// Try to read the file from http postback
		sql := conv.CaseSnake(fld.Name)
		f, fh, err := req.FormFile(sql)
		if err != nil { // => do not try to save this
			log.Error("unable to read attached file", "field", sql)
			delete(post, sql)
			continue
		}

		// Save the file attached on disk
		m, err := NewMedia(f, fh, Table(model), sql, WhoMap(req))
		if err != nil {
			return err
		}

		// Update posted variable
		if m != nil {
			f := m.File()
			b, err := json.Marshal(f)
			if err != nil {
				return err
			}
			post[sql] = string(b)
		}
	}

	return nil
}
