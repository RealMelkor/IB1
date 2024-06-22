//go:build !cgo
package db

import (
	"gorm.io/gorm"
	"github.com/glebarez/sqlite"
)

func sqlite_open(path string) gorm.Dialector {
	return sqlite.Open(Path)
}
