package dorm

import (
	"fmt"
	"strings"
)

const historyPrefix = "zoom_"

func init() {

	// initialize behaviours
	initStaticBehaviors()
	initDynamicBehaviors()
}

func initStaticBehaviors() {

	// UID10
	behave[UID10{}] = []string{
		`CREATE TRIGGER <<Table>>_uid10_bfr_insert BEFORE INSERT ON <<Table>> FOR EACH ROW
		IF NEW.uid IS NULL OR NEW.uid = '' THEN 
			SET @id = 1;
			WHILE (@id IS NOT NULL) DO
				SET NEW.uid = randstr(10);
				SET @id = (SELECT id FROM <<Table>> WHERE uid = NEW.uid);
			END WHILE;
		END IF;`,
	}

	// SoftDelete
	behave[SoftDelete{}] = []string{
		// do not allow delete action
		`CREATE TRIGGER <<Table>>_softdelete_bfr_delete BEFORE DELETE ON <<Table>> FOR EACH ROW
		IF TRUE THEN 
			SIGNAL SQLSTATE '45000'
			SET MESSAGE_TEXT = 'Cannot delete records from table. Instead set deleted=1';
		END IF;`,
		// update deleted_at timestamp
		`CREATE TRIGGER <<Table>>_softdelete_bfr_update BEFORE UPDATE ON <<Table>> FOR EACH ROW
        BEGIN
            IF (OLD.deleted = 0) AND (NEW.deleted = 1) THEN
                SET NEW.deleted_at = NOW();
            END IF;
            IF (OLD.deleted = 1) AND (NEW.deleted = 0) THEN
                SET NEW.deleted_at = NULL;
            END IF;
        END`,
	}

}

func initDynamicBehaviors() {

	// Timed
	behaveModel[Timed{}] = func(model interface{}) []string {
		type Field struct {
			Name    string  `gorm:"column:Field"`
			Type    string  `gorm:"column:Type"`
			Null    string  `gorm:"column:Null"`
			Key     string  `gorm:"column:Key"`
			Default *string `gorm:"column:Default"`
			Extra   string  `gorm:"column:Extra"`
		}
		var f Field
		err := db().Raw("show columns from " + tableName(model) + " where Field = 'updated_at'").Find(&f).Error
		if err != nil {
			panic(err)
		}
		if !strings.Contains(strings.ToLower(f.Extra), "on update current_timestamp") {
			return []string{
				"ALTER TABLE <<Table>> MODIFY COLUMN updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP",
			}
		}
		return []string{}
	}

	// SEO
	behaveModel[Seo{}] = func(model interface{}) []string {
		s := Seo{}
		return []string{
			`CREATE TRIGGER <<Table>>_seo_bfr_update BEFORE UPDATE ON <<Table>> FOR EACH ROW
			BEGIN
				IF NEW.url = '' THEN
					SIGNAL SQLSTATE '45000'
					SET MESSAGE_TEXT = '<<Table>>.Url cannot be updated to EMPTY';
				END IF;
				IF LEFT(NEW.url,1) <> '/' THEN
					SET NEW.url = CONCAT('/', NEW.url);
				END IF;
				IF (OLD.url <> '') AND (NEW.url <> OLD.url) THEN
					IF NEW.url_past IS NULL THEN
						SET NEW.url_past = JSON_ARRAY();
					END IF;
					IF JSON_CONTAINS(NEW.url_past, JSON_ARRAY(OLD.url)) = 0 THEN
						SET NEW.url_past = JSON_ARRAY_APPEND(NEW.url_past, "$", OLD.url);
					END IF;
				END IF;
			END`,
			fmt.Sprintf(`CREATE TRIGGER <<Table>>_seo_bfr_insert BEFORE INSERT ON <<Table>> FOR EACH ROW
				BEGIN
					DECLARE tmp VARCHAR(256);
					DECLARE count INT DEFAULT 0;
					DECLARE found INT DEFAULT 0;
		
					IF NEW.url = '' THEN
						SET tmp = geturl(NEW.%s);
						SET NEW.url = CONCAT('%s/', tmp);
					END IF;
					IF LEFT(NEW.url,1) <> '/' THEN
						SET NEW.url = CONCAT('/', NEW.url);
					END IF;
		
					SET found = (SELECT COUNT(*) FROM <<Table>> WHERE url = NEW.url);
					WHILE found > 0 DO
						SET count = count + 1;
						IF NOT EXISTS (SELECT * FROM <<Table>> WHERE url = CONCAT(NEW.url,count)) THEN
							SET NEW.url = CONCAT(NEW.url,count);
							SET found = 0;
						END IF;
					END WHILE;
				END`, s.UrlColumn(model), s.UrlPrefix(model)),
		}
	}

}
