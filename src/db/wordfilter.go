package db

import (
	"gorm.io/gorm"
	"regexp"
)

type Wordfilter struct {
	gorm.Model
	CRUD[Wordfilter]
	From		string		`gorm:"unique"`
	To		string
	Disabled	bool
	Regexp		*regexp.Regexp	`gorm:"-:all"`
}
