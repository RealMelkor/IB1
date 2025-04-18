package db

import (
	"errors"
	"html/template"
	"gorm.io/gorm"

	"IB1/config"
)

type Thread struct {
	gorm.Model
	Title		string
	BoardID		int
	Board		Board
	Posts		[]Post
	Alive		bool
	Pinned		bool
	Number		int
	Replies		int `gorm:"-:all"`
	Images		int `gorm:"-:all"`
}

type Board struct {
	gorm.Model
	Name		string `gorm:"unique"`
	LongName	string
	Description	string
	Threads		[]Thread
	Members		[]Membership
	Posts		int
	Disabled	bool
	ReadOnly	bool
	Private		bool
	CountryFlag	bool
	PosterID	bool
	OwnerID		*uint
	Owner		Account
}
var Boards map[string]Board

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
		"ORDER BY a.pinned DESC, MAX(c.timestamp) DESC LIMIT ?;",
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

func refreshBoard(board *Board, limit uint) error {
	board.Threads = []Thread{}
	err := db.Raw(
		"SELECT b.* FROM posts a " +
		"INNER JOIN threads b ON a.thread_id = b.id " +
		"WHERE a.board_id = ? AND (a.sage IS NULL OR a.sage <> 1) " +
		"GROUP BY a.thread_id " +
		"ORDER BY b.pinned DESC, MAX(a.timestamp) DESC LIMIT ?;",
		board.ID, limit).
		Scan(&board.Threads).Error
	if err != nil { return err }
	for i := range board.Threads {
		board.Threads[i].Board = *board
	}
	return db.Model(Board{}).Preload("Owner").Find(board).Error;
}

func RefreshBoard(board *Board) error {
	return refreshBoard(board, config.Cfg.Board.MaxThreads)
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

func (thread *Thread) Pin() error {
	thread.Pinned = !thread.Pinned
	return db.Model(&Thread{}).Where("id = ?", thread.ID).
			Update("Pinned", thread.Pinned).Error
}

func RefreshThread(thread *Thread) error {
	return db.Model(*thread).Preload("Posts").Find(thread).Error;
}

func CreateBoard(name string, longName string,
			description string, ownerID uint) error {
	var board Board
	if err := db.First(&board, "Name = ?", name).Error; err != nil {
		ret := db.Create(&Board{Name: name,
				Description: description,
				LongName: longName,
				OwnerID: &ownerID,
			})
		if ret.Error == nil { return ret.Error }
		if ret.Find(&board).Error != nil { return ret.Error }
	}
	Boards[name] = board
	return nil
}

func DeleteThreads(board Board) error {
	maxThreads := config.Cfg.Board.MaxThreads
	if maxThreads == 0 { return nil }
	if err := refreshBoard(&board, ^uint(0)); err != nil { return err }
	if uint(len(board.Threads)) <= maxThreads { return nil }
	threads := board.Threads[maxThreads:len(board.Threads)]
	for _, v := range threads {
		err := Remove(v.Board.Name, int(v.Number))
		if err != nil { return err }
	}
	return nil
}

func CreateThread(board Board, title string, name string, media string,
		ip string, session string, account Account, signed bool,
		rank bool, content template.HTML) (int, error) {
	number := -1
	err := db.Transaction(func(tx *gorm.DB) error {
		var err error
		thread := &Thread{Board: board, Title: title, Alive: true}
		ret := tx.Create(thread)
		if ret.Error != nil { return ret.Error }
		if err := ret.Find(&thread).Error; err != nil { return err }
		number, err = CreatePost(*thread, content, name, media, ip,
				session, account, signed, rank, false, tx)
		if err != nil { return err }
		err = tx.Model(thread).Update("Number", number).Error
		return err
	})
	if err == nil { err = DeleteThreads(board) }
	return number, err
}

func UpdateBoard(board Board) error {
	v := board.OwnerID
	err := db.Save(&board).Error
	if err != nil { return err }
	if v != nil { return nil }
	return db.Model(&board).Select("owner_id").Updates(
		map[string]interface{}{"owner_id": nil}).Error
}

func DeleteBoard(board Board) error {
	return db.Unscoped().Delete(&board).Error
}

func GetBoards() ([]Board, error) {
	var boards []Board
	return boards, db.Preload("Owner").Find(&boards).Error
}

func (board Board) GetMembers() []Membership {
	var members []Membership
	db.Model(&Membership{}).Preload("Member").Preload("Rank").
		Where("board_id = ?", board.ID).Find(&members)
	return members
}

func (board Board) GetMember(account Account) (Membership, error) {
	var member Membership
	i := db.Model(&Membership{}).Preload("Member").Preload("Rank").
		Where("board_id = ? AND member_id = ?", board.ID, account.ID).
		Find(&member).RowsAffected
	if i < 1 {
		return Membership{}, errors.New("not a member")
	}
	return member, nil
}

func (board Board) RemoveMember(name string) error {
	acc, err := GetAccount(name)
	if err != nil { return err }
	return db.Where("board_id = ? AND member_id = ?", board.ID, acc.ID).
		Unscoped().Delete(&Membership{}).Error
}

func (board Board) AddMember(name string, rank string) error {
	acc, err := GetAccount(name)
	if err != nil { return err }
	memberRank, err := GetMemberRank(rank)
	if err != nil { return err }
	return Membership{}.Add(Membership{
		Board: board, Member: acc, Rank: memberRank})
}

func (board Board) UpdateMember(name string, rank string) error {
	acc, err := GetAccount(name)
	if err != nil { return err }
	memberRank, err := GetMemberRank(rank)
	if err != nil { return err }
	return db.Model(Membership{}).
		Where("board_id = ? AND member_id = ?", board.ID, acc.ID).
		Update("rank_id", memberRank.ID).Error
}
