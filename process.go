package dorm

type Process struct {
	PKey

	Name string `sql:"TYPE:varchar(96);not null;" json:"process" insert:"must" unique:"true"`

	LockID string `sql:"TYPE:varchar(64);not null;" json:"lock_id" unique:"true" insert:"must"`
	Locked uint8  `sql:"TYPE:tinyint(1) unsigned;not null;DEFAULT:'0'" json:"acquired"`

	Error *string `sql:"TYPE:varchar(96);" json:"error"`
	Notes *string `sql:"TYPE:varchar(128);" json:"notes"`

	Historic
	WhosThat
	Timed
}
