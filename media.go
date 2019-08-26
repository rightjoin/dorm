package dorm

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif" // necessary image formats
	_ "image/jpeg"
	_ "image/png"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rightjoin/fig"
)

type Media struct {
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

func NewMedia(f multipart.File, fh *multipart.FileHeader, entity, field string, who map[string]interface{}) (*Media, error) {

	var md Media

	// Close f (multipart file) eventually
	defer f.Close()

	// File size
	var buf bytes.Buffer
	fsize, err := buf.ReadFrom(f)
	md.Size = uint(fsize)

	// File type
	md.Mime = http.DetectContentType(buf.Bytes())

	// File extension
	md.Extn = filepath.Ext(fh.Filename)[1:]

	// Filename:
	// remove spaces and better yet, all non-alphanumeric
	// characters from the filename. keeps it simple and avoids
	// some errors
	reg, _ := regexp.Compile("[^a-zA-Z0-9-_/.]+")
	md.Name = reg.ReplaceAllString(fh.Filename, "")

	// additional properties
	md.Entity = entity
	md.Field = field
	md.Who = NewJDoc2(who)

	// Dimensions (if it is an image)
	if strings.HasPrefix(md.Mime, "image/") {
		img, _, err := image.DecodeConfig(bytes.NewReader(buf.Bytes()))
		if err == nil {
			md.Width = &img.Width
			md.Height = &img.Height
		}
	}

	// Validation for size & dimensions
	if err := md.ValidateSize(); err != nil {
		return nil, err
	}

	// Save to db (first)
	dbo := GetORM(true)
	err = dbo.Create(&md).Error
	if err != nil {
		return nil, err
	}
	dbo.Where("id=?", md.ID).Find(&md)

	// Save bytes to disk (second)
	path, err := md.DiskWrite(buf.Bytes(), false)
	if err != nil {
		return nil, err
	}

	// Upload to S3
	uploadToS3 := fig.BoolOr(false, "media.s3.upload")
	if uploadToS3 {
		if err = UploadToS3(buf.Bytes(), path, md.Mime, fsize); err != nil {
			return nil, err
		}
	}

	return &md, nil
}

func (f Media) BeforeCommit() error {
	return f.ValidateSize()
}

// ValidateSize checks the file size and file dimensions
// of the given media against the desired configuration values provided.
// Allowed configurations include:
//    media.validations.max-kb (10*1024 = 10MB default)
//    media.validations.product.max-kb
//    media.validations.product.rear-view.max-kb
//    media.validations.product.rear-view.exact-w
//    media.validations.product.rear-view.min-w
// Overall default max size the system assumes if there is
// no configuration provided is 10MB.
func (f Media) ValidateSize() error {
	prefix := "media.validations"
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
		maxW := fig.IntOr(0, prefix, f.Entity, f.Field, "max-w")
		maxH := fig.IntOr(0, prefix, f.Entity, f.Field, "max-h")

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

		// max dimension checks
		if maxW != 0 && *f.Width > maxW {
			return fmt.Errorf("max width %d, found %d in %s", maxW, *f.Width, f.Name)
		}
		if maxH != 0 && *f.Height > maxH {
			return fmt.Errorf("max height %d, found %d in %s", minH, *f.Height, f.Name)
		}
	}

	// all good
	return nil
}

// DiskWrite writes the content of file passed as input
// parameter to the correct folder on disk.
// If the flag temp is set, the file gets written in to a temp directory.
func (f Media) DiskWrite(raw []byte, temp bool) (string, error) {

	// path at which the files are found
	directory := fig.StringOr("./media", "media.folder")
	if !strings.HasSuffix(directory, "/") {
		directory += "/"
	}
	if temp {
		tempFile, err := ioutil.TempFile("", "video")
		if err != nil {
			return "", nil
		}

		if _, err := tempFile.Write(raw); err != nil {
			return "", err
		}

		return tempFile.Name(), nil
	}

	directory += f.Folder()

	// create nested folders
	err := os.MkdirAll(directory, 0755)
	if err != nil {
		return "", err
	}

	// touch file on disk
	path := fmt.Sprintf("%s/%s", directory, f.Name)
	out, err := os.Create(path)
	if err != nil {
		return "", err
	}

	// write content to file on disk
	_, err = io.Copy(out, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}

	return path, nil
}

// Folder retrieves the path at which the file is
// saved on disk. The path consists for following portions:
//	  prefix : optional prefix, picked up from config @ "media.path-prefix"
//    entity : which refers to this media
//    field  : column of that entity which refers to this media
//    UID    : uid is broken into 3 sub-folders, each of length 3, 3, and 4 chars
func (f Media) Folder() string {

	prefix := fig.StringOr("", "media.path-prefix")
	if prefix != "" {
		prefix += "/"
	}
	part1 := f.UID[0:3]
	part2 := f.UID[3:6]
	part3 := f.UID[6:]

	return fmt.Sprintf("%s%s/%s/%s/%s", prefix, f.Entity, part1, part2, part3)
}

func (f Media) URL() string {
	return fmt.Sprintf("%s/%s", f.Folder(), f.Name)
}

func (f Media) File() File {

	ref := f.UID
	if FileRef == "ID" {
		ref = fmt.Sprintf("%d", f.ID)
	}

	return File{
		Ref:    ref,
		Src:    f.URL(),
		Mime:   f.Mime,
		Width:  f.Width,
		Height: f.Height,
	}
}
