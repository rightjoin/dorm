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
	Ref    string `json:"ref"`
	Src    string `json:"src"`
	Mime   string `json:"mime"`
	Width  *int   `json:"width"`
	Height *int   `json:"height"`
}

type Files []File

// SaveAndGetAnyFile does what SaveAnyFile does. But it also
// additionally returns any dorm.File object created
func SaveAndGetAnyFile(req *http.Request, post map[string]string, model interface{}) ([]File, error) {

	var files []File

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
			return files, err
		}

		// Update posted variable
		if m != nil {
			f := m.File()
			if files == nil {
				files = []File{f}
			} else {
				files = append(files, f)
			}
			b, err := json.Marshal(f)
			if err != nil {
				return files, err
			}
			post[sql] = string(b)
		}
	}

	return files, nil
}

// SaveAnyFile first traverses all the fields of a given model. If it
// finds any dorm.Img field, then it tries to see if a file is part of
// post data. If it finds any such files, then it saves them to disk
// and updates the post data, so that appropriate references be stored
// in the corresponding entity table
func SaveAnyFile(req *http.Request, post map[string]string, model interface{}) error {
	_, e := SaveAndGetAnyFile(req, post, model)
	return e
}

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

func (p *Files) Value() (driver.Value, error) {
	if p == nil {
		return nil, nil
	}
	str, err := json.Marshal(p)
	return string(str), err
}

func (p *Files) Scan(value interface{}) error {
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
