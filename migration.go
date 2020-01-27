package dorm

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rightjoin/fig"
	log "github.com/rightjoin/rlog"
	rip "github.com/rightjoin/rutl/ip"
)

type SQLMigrationTask struct {
	PKey

	Filename string `sql:"VARCHAR(128);not null" unique:"true" insert:"must" update:"false" json:"filename"`

	// Set it to 1, incase you need to run the sql from the same file again.
	// Reduces the need for creating/renaming a file incase of any syntactical errors.
	ReRun uint `sql:"tinyint(1);unsigned;DEFAULT:0" json:"re_run"`

	Remarks string `sql:"varchar(256)" json:"remarks"`

	// Behaviours
	WhosThat
	Timed4
}

const (
	sqlMigrationRLogMessage = "Running SQL Migration"
)

// RunMigration kicks off the migration task; picking and executing sql files present
// inside the directory mentioned in the config under the key
// sql-migration:
//		dir: # defaults to ./migration
func RunMigration() {

	dir := fig.StringOr("./migration", "sql-migration.dir")
	fInfo := make([]os.FileInfo, 0)

	// Collect all the files present inside the migration directory
	err := filepath.Walk(dir, visit(&fInfo))
	if err != nil {
		return
	}
	if len(fInfo) == 0 {
		log.Info(sqlMigrationRLogMessage, "NO sql files found under dir ", dir)
		return
	}

	// sort @modified date
	sort.Slice(fInfo, func(i, j int) bool {
		return fInfo[i].ModTime().Before(fInfo[j].ModTime())
	})

	files := make([]string, len(fInfo))
	for i, val := range fInfo {
		files[i] = val.Name()
	}

	filesToExecute, err := getFilesToExecute(files)
	if err != nil {
		log.Error(sqlMigrationRLogMessage, "Error", err)
		return
	}

	if len(filesToExecute) == 0 {
		log.Info(sqlMigrationRLogMessage, "Msg", "No new files found")
		return
	}

	otp := func(min, max int) string {
		return strconv.Itoa(rand.Intn(max-min) + min)
	}

	db := GetORM(true)
	for _, file := range filesToExecute {

		code := otp(1000, 10000)
		path := fmt.Sprintf("%s/%s", dir, file)
		data, err := ioutil.ReadFile(path)
		if err != nil {
			log.Info(sqlMigrationRLogMessage, "File", path, "Error", err)
			continue
		}

		query := string(data)
		skipFile := false

		log.Info(sqlMigrationRLogMessage, "file-name", file, "Executing Query", query)
		if strings.Contains(strings.ToLower(query), "delete") {

			fmt.Println("Restricted keyword DELETE found in file: " + file)

			reader := bufio.NewReader(os.Stdin)
			fmt.Println("Are you sure you want to continue ?? ( Type:", code, ")")
			input, _ := reader.ReadString('\n')

			if strings.TrimSpace(input) != code {
				fmt.Println("Incorrect entry. Ignoring file " + file)

				skipFile = true
				err = errors.Errorf("skipping file %s, contains restricted keyword DELETE")
			}
		}

		if !skipFile {
			err = db.Exec(query).Error
		}

		if err != nil {
			log.Info(sqlMigrationRLogMessage, "file-name", file, "Execution Status", "Failed", "Error", err)
		} else {
			log.Info(sqlMigrationRLogMessage, "file-name", file, "Execution Status", "Success")
		}

		who := map[string]interface{}{
			"MacAddress": macUint64(),
			"IP":         rip.GetLocal(),
		}

		s := SQLMigrationTask{
			Filename: file,
			ReRun:    0,
			WhosThat: WhosThat{Who: NewJDoc2(who)},
			Timed4: Timed4{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Remarks: "Success",
		}

		if err != nil {
			s.Remarks = err.Error()
		}

		err = db.Where("filename = ?", file).FirstOrCreate(&s).Assign(SQLMigrationTask{ReRun: 0, Timed4: Timed4{UpdatedAt: time.Now()}, Remarks: s.Remarks, WhosThat: s.WhosThat}).Error
		if err != nil {
			log.Info(sqlMigrationRLogMessage, "File", path, "Update Status Failed", err)
		}
	}

}

// getFilesToExecute takes in the list of files that are present in the migration
// directory and checks which of the files have already been run or which of them
// is required to be run again and returns such files list.
func getFilesToExecute(files []string) ([]string, error) {
	db := GetORM(true)

	isPresent := db.HasTable(&SQLMigrationTask{})
	if !isPresent {
		db.CreateTable(&SQLMigrationTask{})

		// Since, the environment seems new, all the files present need
		// to be executed.
		return files, nil
	}

	executedTasks := []SQLMigrationTask{}

	err := db.Where("filename IN (?) OR re_run = 1", files).Find(&executedTasks).Error
	if err != nil {
		return nil, errors.Wrapf(err, "Error while fetching the list of already executed files")
	}

	if len(executedTasks) == 0 {
		return files, nil
	}

	newFiles := []string{}
	aleadyRunMap := make(map[string]bool)
	reRun := make(map[string]bool)

	for _, val := range executedTasks {
		aleadyRunMap[val.Filename] = true
		if val.ReRun == 1 {
			reRun[val.Filename] = true
		}
	}

	// collect either the new files or the files to be re_run
	for _, file := range files {
		if _, ok := aleadyRunMap[file]; !ok {
			newFiles = append(newFiles, file)
			continue
		}
		if _, ok := reRun[file]; ok {
			newFiles = append(newFiles, file)
		}
	}

	return newFiles, nil
}

func visit(files *[]os.FileInfo) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {

		if err != nil {
			log.Error(sqlMigrationRLogMessage, "Error", err)
			return err
		}

		if !info.IsDir() && filepath.Ext(path) == ".sql" {
			*files = append(*files, info)
		}
		return nil
	}
}

func macUint64() uint64 {
	interfaces, err := net.Interfaces()
	if err != nil {
		return uint64(0)
	}

	for _, i := range interfaces {
		if i.Flags&net.FlagUp != 0 && bytes.Compare(i.HardwareAddr, nil) != 0 {

			// Skip locally administered addresses
			if i.HardwareAddr[0]&2 == 2 {
				continue
			}

			var mac uint64
			for j, b := range i.HardwareAddr {
				if j >= 8 {
					break
				}
				mac <<= 8
				mac += uint64(b)
			}

			return mac
		}
	}

	return uint64(0)
}
