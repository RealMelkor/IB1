package db

import (
	"gorm.io/gorm"
	"gorm.io/driver/sqlite"
	"gorm.io/driver/mysql"
	"html/template"
	"errors"
	"IB1/config"
)

const DefaultName = "Anonymous"

type Board struct {
	gorm.Model
	Name		string `gorm:"unique"`
	LongName	string
	Description	string
	Threads		[]Thread
	Posts		int
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
	Title		string `gorm:"-:all"`
}

type Reference struct {
	gorm.Model
	From		int
	To		int
}

const (
	TYPE_SQLITE = iota
	TYPE_MYSQL
)

var db *gorm.DB
var dbType int

func Init() error {

	Boards = map[string]Board{}

	switch config.Cfg.Database.Type {
	case "mysql":
		dbType = TYPE_MYSQL
	case "sqlite":
		dbType = TYPE_SQLITE
	default:
		return errors.New("unknown database " +
				config.Cfg.Database.Type)
	}

	var err error
	if dbType == TYPE_MYSQL {
		dsn := config.Cfg.Database.Url
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	} else if dbType == TYPE_SQLITE {
		db, err = gorm.Open(sqlite.Open(config.Cfg.Database.Url),
				&gorm.Config{})
	} else {
		return errors.New("unknown database")
	}
	if err != nil { return err }

	db.AutoMigrate(&Board{}, &Thread{}, &Post{})

	for _, v := range config.Cfg.Boards {
		if !v.Enabled { continue }
		err := CreateBoard(v.Name, v.Title, v.Description)
		if err != nil { return err }
	}

	return nil
}

func GetBoard(name string) (Board, error) {
	var board Board
	if err := db.First(&board, "Name = ?", name).Error; err != nil {
		return Board{}, err
	}
	if err := RefreshBoard(&board); err != nil {
		return Board{}, err
	}
	return board, nil
}

func RefreshBoard(board *Board) error {
	err := db.Raw(
		"SELECT b.* FROM posts a " +
		"INNER JOIN threads b ON a.thread_id = b.id " +
		"WHERE a.board_id = ? GROUP BY a.thread_id " +
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
	if err := db.Model(*thread).Preload("Posts").Find(thread).Error;
			err != nil {
		return err
	}
	return nil
}

func GetBoards() ([]Board, error) {
	var boards []Board
	err := db.Find(&boards).Error
	return boards, err
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
		content template.HTML) (int, error) {
	number := -1
	err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		thread := &Thread{Board: board, Title: title, Alive: true}
		ret := tx.Create(thread)
		if ret.Error != nil { return ret.Error }
		if err := ret.Find(&thread).Error; err != nil { return err }
		number, err = CreatePost(*thread, content, name, media, tx)
		if err != nil { return err }
		err = tx.Model(thread).Update("Number", number).Error
		return err
	})
	return number, err
}
