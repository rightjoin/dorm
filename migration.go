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

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/rightjoin/fig"
	rip "github.com/rightjoin/rutl/ip"
	"github.com/rightjoin/slog"
)

type SQLMigration struct {
	PKey

	Filename string `sql:"varchar(128);not null" unique:"true" insert:"must" update:"false" json:"filename"`

	// Set it to 1, incase you need to run the sql from the same file again.
	// Reduces the need for creating/renaming a file incase of any syntactical errors.
	ReRun uint `sql:"tinyint(1);unsigned;DEFAULT:0" json:"re_run"`

	Success uint   `sql:"tinyint(1);unsigned;DEFAULT:0" json:"success"`
	Message string `sql:"varchar(512);DEFAULT:''" json:"message"`

	// Behaviours
	Historic
	WhosThat
	Timed
}

// RunMigration kicks off the migration task; picking and executing sql files present
// inside the directory mentioned in the config under the key
// database:
//		sql-folder: defaults to ./sql
func RunMigration(db *gorm.DB) {

	dir := fig.StringOr("./sql", "database.sql-folder")
	fInfo := make([]os.FileInfo, 0)

	// Collect all the files present inside the migration directory
	err := filepath.Walk(dir, visit(&fInfo))
	if err != nil {
		return
	}
	if len(fInfo) == 0 {
		slog.Info("sql: No scripts found", "folder", dir)
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

	filesToExecute, err := getFilesToExecute(db, files)
	if err != nil {
		slog.Error("sql: Error detecting scripts to execute", "error", err)
		return
	}

	if len(filesToExecute) == 0 {
		slog.Info("sql: No (new) scripts to execute :)")
		return
	}

	genOtp := func(min, max int) string {
		return strconv.Itoa(rand.Intn(max-min) + min)
	}

	for _, file := range filesToExecute {

		path := fmt.Sprintf("%s/%s", dir, file)
		data, err := ioutil.ReadFile(path)
		if err != nil {
			slog.Error("sql: Unable to read file", "filepath", path, "error", err)
			continue
		}

		queries := string(data)
		skipFile := false

		slog.Info("sql: Processing...", "filepath", path, "data", queries)

		// Does the file contain restricted keywords?
		restricted := []string{"delete", "drop"}
		containsRestricted := func() bool {
			out := false
			queriesLower := strings.ToLower(queries)
			for _, keyword := range restricted {
				out = strings.Contains(queriesLower, keyword)
				if out {
					return out
				}
			}
			return out
		}()

		// If yes, then get explicit consent
		if containsRestricted {

			code := genOtp(1000, 10000)
			fmt.Println("Restricted keywords found in file: ", file, ". To continue enter:", code)

			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n')

			if strings.TrimSpace(input) != code {
				fmt.Println("Incorrect entry. Skipping file: " + file)

				skipFile = true
				err = errors.Errorf("Skipped file %s due to OTP mismatch", file)
			}
		}

		if !skipFile {
			err = db.Exec(queries).Error
		}

		status := "Success"
		statusInt := 1
		if err != nil {
			status = "Failed"
			statusInt = 0
		}
		slog.Info("sql: Execution status", "filename", file, "status", status, "error", err)

		errDb := db.Where("filename = ?", file).Find(&SQLMigration{}).Error
		rowExists := false
		if errDb == nil {
			rowExists = true
		}

		message := ""
		if err != nil {
			message = err.Error()
		}

		whoData := WhoProc("sql migration", "mac_address", macUint64(), "ip", rip.GetLocal())

		if rowExists {
			// Update
			errDb = Update(db, "filename", file, &SQLMigration{}, "re_run", 0, "success", statusInt, "message", message, "who", whoData)
		} else {
			// Insert
			errDb = Insert(db, &SQLMigration{}, "filename", file, "re_run", 0, "success", statusInt, "message", message, "who", whoData)
		}

		if errDb != nil {
			slog.Error("sql: DB update failed", "filepath", path, "error", errDb, "exists", rowExists)
		}
	}
}

// getFilesToExecute takes in the list of files that are present in the migration
// directory and checks which of the files have already been run or which of them
// is required to be run again and returns such files list.
func getFilesToExecute(db *gorm.DB, files []string) ([]string, error) {

	isPresent := db.HasTable(&SQLMigration{})
	if !isPresent {
		// Since, the environment seems new, all the files present need
		// to be executed.
		return files, nil
	}

	executedTasks := []SQLMigration{}

	err := db.Where("filename IN (?) OR re_run = 1", files).Find(&executedTasks).Error
	if err != nil {
		return nil, errors.Wrapf(err, "Error while fetching the list of already executed files")
	}

	if len(executedTasks) == 0 {
		return files, nil
	}

	newFiles := []string{}
	alreadyRunMap := make(map[string]bool)
	reRun := make(map[string]bool)

	for _, val := range executedTasks {
		alreadyRunMap[val.Filename] = true
		if val.ReRun == 1 {
			reRun[val.Filename] = true
		}
	}

	// collect either the new files or the files to be re_run
	for _, file := range files {
		if _, ok := alreadyRunMap[file]; !ok {
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
			slog.Error("sql: Cannot visit files", "error", err)
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
