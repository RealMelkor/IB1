//go:build cgo
package db

import (
	"gorm.io/gorm"
	"gorm.io/driver/sqlite"
)

func sqlite_open(path string) gorm.Dialector {
	return sqlite.Open(path)
}
