package db

import (
	"gorm.io/gorm"
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
	Rank		int
	Logged		bool	`gorm:"-:all"`
}

type Session struct {
	AccountID	uint
	Account		Account
	Token		string `gorm:"unique"`
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
	case "sqlite":
		dbType = TYPE_SQLITE
	default:
		return errors.New("unknown database " + Type)
	}

	var err error
	if dbType == TYPE_MYSQL {
		dsn := Path
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	} else if dbType == TYPE_SQLITE {
		db, err = gorm.Open(sqlite_open(Path), &gorm.Config{})
	} else {
		return errors.New("unknown database")
	}
	if err != nil { return err }

	db.AutoMigrate(&Board{}, &Thread{}, &Post{}, &Ban{}, &Theme{},
			&Reference{}, &Account{}, &Session{}, &Config{})

	if err := LoadBoards(); err != nil { return err }
	if err := LoadBanList(); err != nil { return err }
	if err := LoadConfig(); err != nil { return err }

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

func GetBoard(name string) (Board, error) {
	board, ok := Boards[name]
	if !ok { return board, errors.New("board not found") }
	if err := RefreshBoard(&board); err != nil {
		return Board{}, err
	}
	return board, nil
}

func GetVisibleThreads(board Board) ([]Thread, error) {
	var threads []Thread
	err := db.Raw(
		"SELECT a.* FROM threads a " +
		"INNER JOIN posts b ON " +
		"a.number = b.number AND a.id = b.thread_id " +
		"INNER JOIN posts c ON " +
		"a.id = c.thread_id " +
		"WHERE a.board_id = ? AND b.disabled = 0 " +
		"GROUP BY a.id " +
		"ORDER BY MAX(c.timestamp) DESC LIMIT ?;",
		board.ID, config.Cfg.Board.MaxThreads,
	).Order("number").Scan(&threads).Error
	return threads, err
}

func LoadBoards() error {
	var boards []Board
	tx := db.Find(&boards)
	if tx.Error != nil {  return tx.Error }
	Boards = map[string]Board{}
	for _, v := range boards {
		if v.Disabled { continue }
		Boards[v.Name] = v
	}
	return nil
}

func RefreshBoard(board *Board) error {
	board.Threads = []Thread{}
	hide := ""
	err := db.Raw(
		"SELECT b.* FROM posts a " +
		"INNER JOIN threads b ON a.thread_id = b.id " +
		"WHERE a.board_id = ? " + hide + "GROUP BY a.thread_id " +
		"ORDER BY MAX(a.timestamp) DESC LIMIT ?;",
		board.ID, config.Cfg.Board.MaxThreads).
		Scan(&board.Threads).Error
	if err != nil { return err }
	for i := range board.Threads {
		board.Threads[i].Board = *board
	}
	return nil
}

func GetThread(board Board, number int) (Thread, error) {
	var thread Thread
	ret := db.First(&thread, "board_id = ? AND number = ?",
			board.ID, number)
	if ret.Error != nil { return Thread{}, ret.Error }
	if err := RefreshThread(&thread); err != nil { return Thread{}, err }
	thread.Board = board
	return thread, nil
}

func RefreshThread(thread *Thread) error {
	return db.Model(*thread).Preload("Posts").Find(thread).Error;
}

func CreateBoard(name string, longName string, description string) error {
	var board Board
	if err := db.First(&board, "Name = ?", name).Error; err != nil {
		ret := db.Create(&Board{Name: name,
				Description: description,
				LongName: longName})
		if ret.Error == nil { return ret.Error }
		if ret.Find(&board).Error != nil { return ret.Error }
	}
	Boards[name] = board
	return nil
}

func CreateThread(board Board, title string, name string, media string,
		ip string, content template.HTML) (int, error) {
	number := -1
	err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		thread := &Thread{Board: board, Title: title, Alive: true}
		ret := tx.Create(thread)
		if ret.Error != nil { return ret.Error }
		if err := ret.Find(&thread).Error; err != nil { return err }
		number, err = CreatePost(*thread, content, name, media, ip, tx)
		if err != nil { return err }
		err = tx.Model(thread).Update("Number", number).Error
		return err
	})
	return number, err
}

func UpdateBoard(board Board) error {
	return db.Save(&board).Error
}

func DeleteBoard(board Board) error {
	return db.Unscoped().Delete(&board).Error
}

func GetBoards() ([]Board, error) {
	var boards []Board
	return boards, db.Find(&boards).Error
}

func AddTheme(name string, content string, disabled bool) error {
	return db.Create(&Theme{
		Name: name, Content: content, Disabled: disabled}).Error
}

func DeleteTheme(name string) error {
	return db.Where("name = ?", name).Delete(&Theme{}).Error
}

func DeleteThemeByID(id int) error {
	return db.Unscoped().Delete(&Theme{}, id).Error
}

func UpdateThemeByID(id int, name string, disabled bool) error {
	return db.Where("id = ?", id).Select("name", "disabled").
		Updates(Theme{Name: name, Disabled: disabled}).Error
}

func GetThemes() ([]Theme, error) {
	var themes []Theme
	err := db.Find(&themes).Error
	return themes, err
}
