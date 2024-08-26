package db

import (
	"errors"
	"html/template"
	"gorm.io/gorm"

	"IB1/config"
)

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