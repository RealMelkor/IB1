//go:build !cgo

package db

import (
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func sqlite_open(path string) gorm.Dialector {
	return sqlite.Open(Path)
}
