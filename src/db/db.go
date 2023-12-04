package db

import (
	"gorm.io/gorm"
	"gorm.io/driver/sqlite"
	"gorm.io/driver/mysql"
	"html/template"
	"errors"
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
	Title	string
	BoardID	int
	Board	Board
	Posts	[]Post
	Alive	bool
	Number	int
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
}

type PostReference struct {
	gorm.Model
}

const (
	TYPE_SQLITE = iota
	TYPE_MYSQL
)

var db *gorm.DB
var dbType int

func Init() error {

	Boards = map[string]Board{}

	dbType = TYPE_SQLITE

	var err error
	if dbType == TYPE_MYSQL {
		dsn := "root:mypassword@tcp(0.0.0.0:3306)/" +
			"test?charset=utf8mb4&parseTime=True&loc=Local"
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	} else if dbType == TYPE_SQLITE {
		db, err = gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	} else {
		return errors.New("Unknown database")
	}
	if err != nil {
		return err
	}

	db.AutoMigrate(&Board{}, &Thread{}, &Post{})

	if err := CreateBoard("a", "fun", "the first board"); err != nil {
		return err
	}
	if err := CreateBoard("b", "random", "the second board"); err != nil {
		return err
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
	if err := db.Model(*board).Preload("Threads").Find(board).Error;
			err != nil {
		return err
	}
	for i := range board.Threads {
		board.Threads[i].Board = *board
	}
	return nil
}

func GetThread(board Board, number int) (Thread, error) {
	var thread Thread
	ret := db.Where(&Thread{Number: number, Board: board}).First(&thread)
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
