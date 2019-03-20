package dorm

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif" // necessary image formats
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rightjoin/fig"
)

type File struct {
	PKey
	UID10

	Name   string `sql:"TYPE:varchar(128);not null;" json:"name" insert:"no" update:"no"`
	Extn   string `sql:"TYPE:varchar(8);not null;" json:"extn" insert:"no" update:"no"`
	Mime   string `sql:"TYPE:varchar(128);not null;" json:"mime" insert:"no" update:"no"`
	Size   uint   `sql:"not null" json:"size" insert:"no" update:"no"`
	Width  *int   `sql:"null" json:"width" insert:"no" update:"no"`
	Height *int   `sql:"null" json:"height" insert:"no" update:"no"`

	// Entity cross-reference for reverse lookups
	Entity string `sql:"TYPE:varchar(128);not null;" json:"entity" insert:"must"`
	Field  string `sql:"TYPE:varchar(128);not null;" json:"field" insert:"must"`

	// Behaviors
	Active1
	SoftDelete
	WhosThat
	Timed
}

func NewFile(f multipart.File, fh *multipart.FileHeader, entity, field string, who map[string]interface{}) (*File, error) {

	var fi File

	// close f (multipart file) eventually
	defer f.Close()

	// file size
	var buf bytes.Buffer
	fsize, err := buf.ReadFrom(f)
	fi.Size = uint(fsize)

	// file type
	fi.Mime = http.DetectContentType(buf.Bytes())

	// file extension
	fi.Extn = filepath.Ext(fh.Filename)[1:]

	// dimenstions (if it is an image)
	if strings.HasPrefix(fi.Mime, "image/") {
		img, _, err := image.DecodeConfig(bytes.NewReader(buf.Bytes()))
		if err == nil {
			fi.Width = &img.Width
			fi.Height = &img.Height
		}
	}

	// filename:
	// remove spaces and better yet, all non-alphanumeric
	// characters from the filename. keeps it simple and avoids
	// some errors
	reg, _ := regexp.Compile("[^a-zA-Z0-9-_/.]+")
	fi.Name = reg.ReplaceAllString(fh.Filename, "")

	// additional properties
	fi.Entity = entity
	fi.Field = field
	fi.Who = NewJDoc2(who)

	// validation for size & dimensions
	if err := fi.ValidateSize(); err != nil {
		return nil, err
	}

	// save to db
	dbo := GetORM(true)
	err = dbo.Create(fi).Error
	if err != nil {
		return nil, err
	}
	dbo.Where("id=?", fi.ID).Find(&fi)

	// save to disk
	err = fi.DiskWrite(buf.Bytes())
	if err != nil {
		return nil, err
	}

	return &fi, nil
}

func (f File) OnSerialize() error {
	return f.ValidateSize()
}

// ValidateSize checks the file size and file dimensions
// of the given file (if it is an image) against the
// desired configuration values provided.
// Allowed configurations include:
//    file.validations.max-kb (10*1024 = 10MB default)
//    file.validations.product.max-kb
//    file.validations.product.rear-view.max-kb
//    file.validations.product.rear-view.exact-w
//    file.validations.product.rear-view.min-w
// Overall default max size the system assumes if there is
// no configuration provided is 10MB.
func (f File) ValidateSize() error {
	prefix := "file.validations"
	maxKB := "max-kb"

	// if validations are disabled, then skip all checks
	if fig.BoolOr(true, prefix, "active") == false {
		return nil
	}

	// validate file size::
	max := fig.IntOr(0, prefix, f.Entity, f.Field, maxKB) // field level max value
	if max == 0 {
		max = fig.IntOr(0, prefix, f.Entity, maxKB) // entity level max value
	}
	if max == 0 {
		max = fig.IntOr(10*1024, prefix, maxKB) // global max value (default to 10MB)
	}
	if f.Size > uint(max*1024) {
		return fmt.Errorf("file max limit of %d kb exceeded. %d b found in %s", max, f.Size, f.Name)
	}

	// validate file dimensions::
	if strings.HasPrefix(f.Mime, "image/") {
		w := fig.IntOr(0, prefix, f.Entity, f.Field, "exact-w")
		h := fig.IntOr(0, prefix, f.Entity, f.Field, "exact-h")
		minW := fig.IntOr(0, prefix, f.Entity, f.Field, "min-w")
		minH := fig.IntOr(0, prefix, f.Entity, f.Field, "min-h")

		// exact dimension checks
		if w != 0 && w != *f.Width {
			return fmt.Errorf("expected width %d, found %d in %s", w, *f.Width, f.Name)
		}
		if h != 0 && h != *f.Height {
			return fmt.Errorf("expected height %d, found %d in %s", h, *f.Height, f.Name)
		}

		// min dimension checks
		if minW != 0 && *f.Width < minW {
			return fmt.Errorf("min width %d, found %d in %s", minW, *f.Width, f.Name)
		}
		if minH != 0 && *f.Height < minH {
			return fmt.Errorf("min height %d, found %d in %s", minH, *f.Height, f.Name)
		}
	}

	// all good
	return nil
}

// DiskWrite writes the content of file passed as input
// parameter to the correct folder on disk.
// The folder is creating by splitting the UID field into
// 3 parts of 3, 3 and 4 characters
func (f File) DiskWrite(raw []byte) error {

	// path at which the files are found
	directory := fig.String("file.folder")
	if !strings.HasSuffix(directory, "/") {
		directory += "/"
	}
	directory += f.Folder()

	// create nested folders
	err := os.MkdirAll(directory, 0755)
	if err != nil {
		return err
	}

	// touch file on disk
	path := fmt.Sprintf("%s/%s", directory, f.Name)
	out, err := os.Create(path)
	if err != nil {
		return err
	}

	// write content to file on disk
	_, err = io.Copy(out, bytes.NewReader(raw))
	if err != nil {
		return err
	}

	return nil
}

func (f File) Folder() string {

	part1 := f.UID[0:3]
	part2 := f.UID[3:6]
	part3 := f.UID[6:]

	return fmt.Sprintf("%s/%s/%s/%s", f.Entity, part1, part2, part3)
}

func (f File) URL() string {
	return fmt.Sprintf("%s/%s", f.Folder(), f.Name)
}
