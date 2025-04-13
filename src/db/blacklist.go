package db

import (
	"gorm.io/gorm"
)

type Blacklist struct {
	gorm.Model
	CRUD[Blacklist]
	ID		uint
	Disabled	bool
	Host		string `gorm:"unique"`
}
