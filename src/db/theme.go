package db

import (
	"gorm.io/gorm"
)

type Theme struct {
	gorm.Model
	CRUD[Theme]
	Content  string
	Name     string `gorm:"unique"`
	Disabled bool
}
