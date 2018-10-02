package dorm

import (
	"reflect"
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/rightjoin/fig"
	"github.com/rightjoin/utila/conv"
	"github.com/rightjoin/utila/refl"
)

// OverrideDB can be used to override the default connection
// that dorm uses for building schema and population.
// It picks up database.master to do these operations, but
// this can be overridden by DBConn variable
var OverrideDB *gorm.DB

// simple & static behaviours
var behave = map[interface{}][]string{}

// dynamic behaviours, wherein trigger definition depends
// upon some model property
var behaveModel = map[interface{}]func(interface{}) []string{}

type triggered interface {
	Triggers() []string
}

type populateRows interface {
	InitialRecords() []interface{}
}

// db provides the gorm db connection on which
// all operations of building schema and population
// are performed
func db() *gorm.DB {
	if OverrideDB != nil {
		return OverrideDB
	}
	return GetORM(true)
}

func BuildSchema(models ...interface{}) {

	// validations
	if db() == nil {
		panic("connection is null. Please specify the DB to populate")
	}

	// migrate (build basic tables)
	for _, model := range models {
		e := db().AutoMigrate(model).Error
		if e != nil {
			panic(e)
		}
	}

	// build history log
	for _, model := range models {
		if refl.ComposedOf(model, Historic{}) {
			setupHistoricAuditLog(model)
		}
	}

	// build unique indexes
	for _, model := range models {
		setupUniqueIndexes(model)
	}

	// build normal indexes
	for _, model := range models {
		setupIndexes(model)
	}

	// build foreign keys
	for _, model := range models {
		setupForeignKeys(model)
	}

	// build custom behaviors
	for _, model := range models {
		setupBehaviors(model)
	}

	// build custom triggers
	for _, model := range models {
		setupCustomTriggers(model)
	}

	// initial records defined in model
	for _, model := range models {
		insertInitialRecords(model)
	}
}

func CreateDatabase(name string) {

	// Don't use master db connection, as it
	// will try to connect to a non-existing database.
	// So replace db name with information_schema
	engine := fig.String("database.master.engine")
	conn := GetCstrConfig(engine, "database.master")
	currentDB := fig.String("database.master.db")
	conn = strings.Replace(conn, currentDB, "information_schema", -1)
	schema := GetORMCstr(engine, conn)

	// Create the new database
	err := schema.Exec("CREATE DATABASE IF NOT EXISTS " + name + " CHARACTER SET utf8 COLLATE utf8_general_ci").Error
	if err != nil {
		panic(err)
	}

	// Switch to new db (from information schema)
	err = schema.Exec("USE " + name).Error
	if err != nil {
		panic(err)
	}

	// Create the needed functions::

	// function: url cleanup
	err = schema.Exec(`CREATE FUNCTION geturl( str VARCHAR(256) ) RETURNS VARCHAR(256)
	BEGIN
		DECLARE i, len SMALLINT DEFAULT 1;
		DECLARE ret VARCHAR(256) DEFAULT '';
		DECLARE c VARCHAR(1);
		DECLARE prev VARCHAR(1);

		SET str = LCASE(TRIM(str));
		SET len = CHAR_LENGTH(str);

		REPEAT
			BEGIN
				SET c = MID( str, i, 1 );
				IF c REGEXP '[[:alnum:]]' OR c IN ('-','_',' ') THEN
					IF c = ' ' THEN
						SET c = '-';
					END IF;
					IF prev = '-' AND c = '-' THEN
						# do nothing
						SET c = '-';
					ELSE
						SET ret=CONCAT(ret,c);
					END IF;
					SET prev = c;
				END IF;
				SET i = i + 1;
			END;
		UNTIL i > len END REPEAT;
		RETURN ret;
	END`).Error
	if err != nil {
		panic(err)
	}

	// function: random string generator
	err = schema.Exec(`CREATE FUNCTION randstr (length SMALLINT(3)) RETURNS varchar(100)
	BEGIN
		SET @returnStr = '';
		SET @allowedChars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
		SET @i = 0;

		WHILE (@i < length) DO
			SET @returnStr = CONCAT(@returnStr, substring(@allowedChars, FLOOR(RAND() * LENGTH(@allowedChars) + 1), 1));
			SET @i = @i + 1;
		END WHILE;

		RETURN @returnStr;
	END`).Error
	if err != nil {
		panic(err)
	}

	// Switch back underlying connection to
	// information_schema (as the connection was opened to it only)
	err = schema.Exec("USE information_schema").Error
	if err != nil {
		panic(err)
	}
}

func DropDatabase(name string) {

	dbo := db()
	dbname := fig.String("database.master.db")
	err := db().Exec("drop database " + dbname).Error
	if err != nil {
		panic(err)
	}

	// Now the connection points to database that
	// has been deleted. so remove it from the list
	// of connections
	dbo.Close() // cleanup
	var match string
	for key, val := range connections {
		if val.DB() == dbo.DB() {
			match = key
		}
	}
	if match != "" {
		delete(connections, match)
	}

}

func PopulateRows(records ...interface{}) {
	for _, row := range records {
		txn := db().Begin()

		err := txn.Create(row).Error
		if err != nil {
			txn.Rollback()
			panic(err)
		}

		txn.Commit()
	}
}

// setupUniqueIndexes uses the following formats to
// create a unique index
// unique:"true"
// unique:"idx_name"
// unique:"idx_name(field1,field2)"
func setupUniqueIndexes(model interface{}) {
	fields := refl.NestedFields(model)
	for i := 0; i < len(fields); i++ {
		fld := fields[i]
		if len(fld.Tag.Get("unique")) > 0 {
			name := fld.Tag.Get("unique")
			if name == "true" { // generate index name
				name = "idx_" + conv.CaseSnake(fld.Name) + "_unique"
			}
			lbrace := strings.Index(name, "(")
			if lbrace == -1 {
				err := db().Model(model).AddUniqueIndex(name, conv.CaseSnake(fld.Name)).Error
				if err != nil {
					panic(err)
				}
			} else {
				fldCsv := name[lbrace+1 : len(name)-1]
				flds := strings.Split(fldCsv, ",")
				for i := range flds {
					flds[i] = strings.TrimSpace(flds[i])
				}
				err := db().Model(model).AddUniqueIndex(name[:lbrace], flds...).Error
				if err != nil {
					panic(err)
				}
			}
		}
	}

}

// setupIndexes uses the following given formats
// to create an index on the underlying table
// index:"true"
// index:"idx_name"
// index:"idx_name(field1,field2)"
func setupIndexes(model interface{}) {
	fields := refl.NestedFields(reflect.ValueOf(model).Elem().Interface())
	for i := 0; i < len(fields); i++ {
		fld := fields[i]
		if len(fld.Tag.Get("index")) > 0 {
			name := fld.Tag.Get("index")
			if name == "true" { // generate index name
				name = "idx_" + conv.CaseSnake(fld.Name)
			}
			lbrace := strings.Index(name, "(")
			if lbrace == -1 {
				err := db().Model(model).AddIndex(name, conv.CaseSnake(fld.Name)).Error
				if err != nil {
					panic(err)
				}
			} else {
				fldCsv := name[lbrace+1 : len(name)-1]
				flds := strings.Split(fldCsv, ",")
				for i := range flds {
					flds[i] = strings.TrimSpace(flds[i])
				}
				err := db().Model(model).AddIndex(name[:lbrace], flds...).Error
				if err != nil {
					panic(err)
				}
			}
		}
	}

}

// setupForeignKeys configures foreign keys in the
// underlying db using the format below
// fk:"table_name(identity_key)"
func setupForeignKeys(model interface{}) {
	modelType := reflect.TypeOf(model).Elem()
	num := modelType.NumField()
	for i := 0; i < num; i++ {
		fld := modelType.FieldByIndex([]int{i})
		tag := fld.Tag
		if len(tag.Get("fk")) > 0 {
			fk := conv.CaseSnake(fld.Name)
			err := db().Model(model).AddForeignKey(fk, tag.Get("fk"), "RESTRICT", "RESTRICT").Error
			if err != nil {
				panic(err)
			}
		}
	}
}

// setupCustomTriggers creates the given triggers.
// The triggers must be specified in the Triggers()
// method, that returns an array of strings
func setupCustomTriggers(model interface{}) {
	if m, ok := model.(triggered); ok {
		triggers := m.Triggers()
		for _, trig := range triggers {
			err := db().Exec(trig).Error
			if err != nil {
				panic(err)
			}
		}
	}
}

func setupBehaviors(model interface{}) {

	tbl := tableName(model)

	// skip behaviours for "zoom_" tables
	if strings.HasPrefix(tbl, "zoom_") {
		return
	}

	exec := func(inp string) {
		inp = strings.Replace(inp, "<<Table>>", tbl, -1)
		err := db().Exec(inp).Error
		if err != nil {
			panic(err)
		}
	}

	// static behaviors
	for obj, triggs := range behave {
		if refl.ComposedOf(model, obj) {
			for _, t := range triggs {
				exec(t)
			}
		}
	}

	// dynamic behaviors
	for obj, fn := range behaveModel {
		if refl.ComposedOf(model, obj) {
			triggs := fn(model)
			for _, t := range triggs {
				exec(t)
			}
		}
	}

}

func setupHistoricAuditLog(model interface{}) {

	tbl := tableName(model)
	hist := historyPrefix + tbl

	exec := func(inp string) {
		inp = strings.Replace(inp, "<<Table>>", hist, -1)
		inp = strings.Replace(inp, "<<TableOrig>>", tbl, -1)
		err := db().Exec(inp).Error
		if err != nil {
			panic(err)
		}
	}

	type Field struct {
		Name    string  `gorm:"column:Field"`
		Type    string  `gorm:"column:Type"`
		Null    string  `gorm:"column:Null"`
		Key     string  `gorm:"column:Key"`
		Default *string `gorm:"column:Default"`
		Extra   string  `gorm:"column:Extra"`
	}
	var flds []Field

	info := func(f Field) string {
		key := f.Name + " " + f.Type
		if f.Null == "NO" {
			key += " NOT NULL"
		}
		if f.Default != nil {
			key += " DEFAULT " + *(f.Default)
		}
		return key + " " + f.Extra
	}

	// create table alike
	exec("CREATE TABLE <<Table>> LIKE <<TableOrig>>;")

	// remove auto increment (if any)
	sql := "SHOW COLUMNS FROM " + hist + " WHERE Extra LIKE '%auto_increment%'"
	err := db().Raw(sql).Find(&flds).Error
	if err != nil {
		panic(err)
	}
	if flds != nil && len(flds) > 0 {
		for _, f := range flds {
			exec("ALTER TABLE <<Table>> MODIFY " + strings.Replace(info(f), "auto_increment", "", -1))
		}
	}

	// drop primary key (if any)
	sql = "SHOW COLUMNS FROM " + hist + " WHERE `Key` = 'PRI'"
	err = db().Raw(sql).Find(&flds).Error
	if err != nil {
		panic(err)
	}
	if flds != nil && len(flds) > 0 {
		exec("ALTER TABLE <<Table>> DROP PRIMARY KEY")
	}

	// add columns: row_id, action and actioned_at
	exec("ALTER TABLE <<Table>> ADD COLUMN row_id bigint unsigned first, ADD COLUMN action varchar(6) not null default 'insert' after row_id, ADD COLUMN actioned_at DATETIME not null default current_timestamp after action")

	// set primary key and auto_increment on row_id
	exec("ALTER TABLE <<Table>> ADD PRIMARY KEY (row_id)")
	exec("ALTER TABLE <<Table>> MODIFY COLUMN row_id BIGINT UNSIGNED auto_increment")

	// setup triggers on original/base table::

	exec("DROP TRIGGER IF EXISTS <<TableOrig>>_audit_trail_insert")
	exec(`CREATE TRIGGER <<TableOrig>>_audit_trail_insert AFTER INSERT ON <<TableOrig>> FOR EACH ROW
        INSERT INTO <<Table>> SELECT null,'insert',NOW(), src.* 
        FROM <<TableOrig>> as src WHERE src.id = NEW.id;`)

	exec("DROP TRIGGER IF EXISTS <<TableOrig>>_audit_trail_update")
	exec(`CREATE TRIGGER <<TableOrig>>_audit_trail_update AFTER UPDATE ON <<TableOrig>> FOR EACH ROW
        INSERT INTO <<Table>> SELECT null,'update',NOW(), src.* 
        FROM <<TableOrig>> as src WHERE src.id = NEW.id;`)

	exec("DROP TRIGGER IF EXISTS <<TableOrig>>_audit_trail_delete")
	exec(`CREATE TRIGGER <<TableOrig>>_audit_trail_delete BEFORE DELETE ON <<TableOrig>> FOR EACH ROW
        INSERT INTO <<Table>> SELECT null,'delete',NOW(), src.* 
        FROM <<TableOrig>> as src WHERE src.id = OLD.id;`)

}

func insertInitialRecords(model interface{}) {
	if m, ok := model.(populateRows); ok {
		recs := m.InitialRecords()
		PopulateRows(recs...)
	}
}
