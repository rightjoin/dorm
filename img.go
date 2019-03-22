package dorm

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	log "github.com/rightjoin/rlog"
	"github.com/rightjoin/rutl/conv"

	"github.com/rightjoin/rutl/refl"
)

var imgSign = "*st:github.com/rightjoin/dorm.Img"

// ImgFileRef global variable controls which of the two references
// (UID or ID) of File is stored for lookups inside Img struct.
var ImgFileRef string = "UID" // Possible value sare UID or ID

type Img struct {
	FileRef string `json:"file_ref"`
	Src     string `json:"src"`
}

type ImgMulti struct {
	Images *ImgList `sql:"TYPE:json" json:"images"`
}

type ImgList []Img

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

func (p *ImgList) Value() (driver.Value, error) {
	if p == nil {
		return nil, nil
	}
	str, err := json.Marshal(p)
	return string(str), err
}

func (p *ImgList) Scan(value interface{}) error {
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

// SaveAnyImg first traverses all the fields of a given model. If it
// finds any dorm.Img field, then it tries to see if a file is part of
// post data. If it finds any such files, then it saves them to disk
// and updates the post data, so that appropriate references be stored
// in the corresponding entity table
func SaveAnyImg(req *http.Request, post map[string]string, model interface{}) error {

	for _, fld := range refl.NestedFields(model) {
		if refl.Signature(fld.Type) != imgSign {
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

		// Save the file on disk
		file, err := NewFile(f, fh, Table(model), sql, WhoMap(req))
		if err != nil {
			return err
		}

		// Update posted variable
		if file != nil {
			var ref = file.UID
			if ImgFileRef == "ID" {
				ref = fmt.Sprintf("%d", file.ID)
			}
			post[sql] = fmt.Sprintf(`{"file_ref":"%s", "src":"%s"}`, ref, file.URL())
		}
	}

	return nil
}
