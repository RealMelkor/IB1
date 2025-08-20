//go:build cgo

package db

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func sqlite_open(path string) gorm.Dialector {
	return sqlite.Open(path)
}
