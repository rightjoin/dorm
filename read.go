package dorm

import (
	"database/sql"
	"fmt"
	"reflect"

	"github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	log "github.com/rightjoin/rlog"
)

func ToMap(dbo *gorm.DB, sql string, params ...interface{}) ([]map[string]interface{}, error) {

	var out []map[string]interface{}

	// Execute the SQL
	rows, err := dbo.Raw(sql, params...).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Column Types
	columns, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	for _, column := range columns {
		fmt.Println(column.Name())
		len, e1 := column.Length()
		null, e2 := column.Nullable()
		prec, scale, e3 := column.DecimalSize()
		log.Debug(column.Name(), "DbTypeName", column.DatabaseTypeName(), "Length", len, "Nullable", null, "gotype", column.ScanType(), "DecimalPrecision", prec, "DecimalScale", scale, "e1", e1, "e2", e2, "e3", e3)
	}

	for rows.Next() {

		// Prepare values to be scanned
		values := make([]interface{}, len(columns)) // Items that will be read
		object := map[string]interface{}{}
		for i, column := range columns {
			val := reflect.New(column.ScanType()).Interface()
			// switch val.(type) {
			// case *[]uint8:
			// 	val = new(string)
			// default:
			// 	log.Info("Unknown type", "column", column.Name(), "type", column.ScanType())
			// }
			log.Info("Unknown type", "column", column.Name(), "type", column.ScanType())

			// Add val for array to be read
			values[i] = val
			object[column.Name()] = val
		}

		// Read values from DB
		err = rows.Scan(values...)
		if err != nil {
			return nil, err
		}

		repurpose(columns, object)
		out = append(out, object)
	}

	return out, nil
}

func repurpose(cols []*sql.ColumnType, data map[string]interface{}) {
	for _, col := range cols {
		key := col.Name()
		val := data[key]
		switch it := val.(type) {
		case *sql.NullString:
			fmt.Println(key, "to string")
			if it.Valid {
				data[key] = it.String
			} else {
				data[key] = nil
			}
		case *sql.NullInt64:
			fmt.Println(key, "to int")
			if it.Valid {
				data[key] = it.Int64
			} else {
				data[key] = nil
			}
		case *mysql.NullTime:
			if it.Valid {
				data[key] = it.Time
			} else {
				data[key] = nil
			}
		default:
			fmt.Println(key, "to", reflect.TypeOf(val))
		}

	}
}
