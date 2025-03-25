package db

import (
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/driver/mysql"
	"errors"
	"os"
	"log"
	"IB1/config"
)

type Config struct {
	gorm.Model
	Data		[]byte
}

const (
	TYPE_SQLITE = iota
	TYPE_MYSQL
)

var db *gorm.DB
var dbType int
var Path string
var Type string

func Init() error {

	sessions.Init()
	Boards = map[string]Board{}

	config.LoadDefault()
	if Type == "" {
		v, ok := os.LookupEnv("DB_TYPE")
		if ok { Type = v }
	}
	if Path == "" {
		v, ok := os.LookupEnv("DB_PATH")
		if ok { Path = v }
	}
	if Type == "" { Type = "sqlite" }
	if Path == "" { Path = "ib1.db" }
	switch Type {
	case "mysql":
		dbType = TYPE_MYSQL
		Path += "?parseTime=true"
	case "sqlite":
		dbType = TYPE_SQLITE
	default:
		return errors.New("unknown database " + Type)
	}

	var err error
	cfg := gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}
	if dbType == TYPE_MYSQL {
		dsn := Path
		db, err = gorm.Open(mysql.Open(dsn), &cfg)
	} else if dbType == TYPE_SQLITE {
		db, err = gorm.Open(sqlite_open(Path), &cfg)
	} else {
		return errors.New("unknown database")
	}
	if err != nil { return err }

	db.AutoMigrate(&Board{}, &Thread{}, &Post{}, &Ban{}, &Theme{},
			&Reference{}, &Account{}, &Session{}, &Config{},
			&Media{}, &Banner{}, &BannedImage{}, &Ban{},
			&Rank{}, &MemberRank{},
			&Wordfilter{}, &CIDR{}, &KeyValue{})

	if err := LoadBoards(); err != nil { return err }
	if err := LoadBanList(); err != nil { return err }
	if err := LoadConfig(); err != nil { return err }
	go func() {
		if err := LoadCountries(); err != nil {
			log.Println(err)
		}
	}()
	if err := UpdateConfig(); err != nil { return err }
	go cleanMediaTask()

	if _, err := GetRank(UNAUTHENTICATED); err != nil {
		defaults := []string{
			CREATE_THREAD.String(),
			CREATE_POST.String(),
		}
		if err := CreateRank(UNAUTHENTICATED, defaults); err != nil {
			return err
		}
	}

	for i := range memberPrivileges {
		memberPrivileges[i] = MemberPrivilege(GetPrivilege(i))
	}

	return nil
}

func newConfig() error {
	config.LoadDefault()
	return UpdateConfig()
}

func LoadConfig() error {
	var cfg Config
	if err := db.First(&cfg).Error; err != nil {
		return newConfig()
	}
	if err := config.LoadConfig(cfg.Data); err != nil {
		return newConfig()
	}
	return nil
}

func UpdateConfig() error {
	var cfg Config
	var err error
	db.Exec("DELETE FROM configs")
	cfg.Data, err = config.GetRaw()
	if err != nil { return err }
	return db.Create(&cfg).Error
}

type CRUD[T any] struct {
	v *T
}

func (CRUD[T]) Add(v T) error {
	return db.Create(&v).Error
}

func (CRUD[T]) Update(id int, v T) error {
	return db.Where("id = ?", id).Select("*").Updates(&v).Error
}

func (CRUD[T]) Get(id int, v T) error {
	return db.Where("id = ?", id).Select("*").First(&v).Error
}

func (CRUD[T]) GetAll() ([]T, error) {
	var v []T
	err := db.Find(&v).Error
	return v, err
}

func (CRUD[T]) Remove(v T) error {
	return db.Unscoped().Delete(&v).Error
}

func (CRUD[T]) RemoveID(id int, v T) error {
	return db.Unscoped().Delete(v, id).Error
}
