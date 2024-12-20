package db

import (
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/driver/mysql"
	"html/template"
	"errors"
	"os"
	"IB1/config"
)

type Config struct {
	gorm.Model
	Data		[]byte
}

type Theme struct {
	gorm.Model
	Content		string
	Name		string `gorm:"unique"`
	Disabled	bool
}

type MediaType uint8
const (
	MEDIA_PICTURE MediaType = iota
	MEDIA_VIDEO
	MEDIA_AUDIO
)

type Media struct {
	Hash		string `gorm:"unique"`
	Mime		string
	Data		[]byte
	Thumbnail	[]byte
	Approved	bool
	Type		MediaType
}

type BannedImage struct {
	gorm.Model
	Hash		int64
	Kind		int
}

type Board struct {
	gorm.Model
	Name		string `gorm:"unique"`
	LongName	string
	Description	string
	Threads		[]Thread
	Posts		int
	Disabled	bool
}
var Boards map[string]Board

type Thread struct {
	gorm.Model
	Title		string
	BoardID		int
	Board		Board
	Posts		[]Post
	Alive		bool
	Number		int
	Replies		int `gorm:"-:all"`
	Images		int `gorm:"-:all"`
}

type Post struct {
	gorm.Model
	Content		template.HTML
	Media		string
	MediaHash	string
	From		string
	Name		string
	ThreadID	int
	Thread		Thread
	BoardID		int
	Board		Board
	Number		int
	Timestamp	int64
	IP		string
	Disabled	bool
	OwnerID		uint
	Owner		Account
	Session		string `gorm:"size:32"`
	Signed		bool
	Rank		string
}

type Reference struct {
	gorm.Model
	From		int
	PostID		int
	ThreadID	int
	Thread		Thread
}

type Account struct {
	gorm.Model
	Name		string	`gorm:"unique"`
	Password	string
	RankID		int
	Rank		Rank
	Logged		bool	`gorm:"-:all"`
	Theme		string
	Superuser	*bool	`gorm:"unique"`
}

type Session struct {
	AccountID	uint
	Account		Account
	Token		string `gorm:"unique"`
}

type Banner struct {
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
			&Media{}, &Banner{}, &BannedImage{}, &Rank{}, &Ban{})

	if err := LoadBoards(); err != nil { return err }
	if err := LoadBanList(); err != nil { return err }
	if err := LoadConfig(); err != nil { return err }
	if err := UpdateConfig(); err != nil { return err }
	go cleanMediaTask()

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
