package db

import (
	"gorm.io/gorm"
	"regexp"
)

type Wordfilter struct {
	gorm.Model
	From		string		`gorm:"unique"`
	To		string
	Disabled	bool
	Regexp		*regexp.Regexp	`gorm:"-:all"`
}

func UpdateWordfilter(id int, from string, to string, disabled bool) error {
	return db.Where("id = ?", id).Select("from", "to", "disabled").
		Updates(&Wordfilter{
			From: from, To: to, Disabled: disabled,
		}).Error
}

func AddWordfilter(from string, to string, disabled bool) error {
	return db.Create(&Wordfilter{
		From: from, To: to, Disabled: disabled}).Error
}

func RemoveWordfilter(id int) error {
	return db.Unscoped().Delete(&Wordfilter{}, id).Error
}

func GetWordfilters() ([]Wordfilter, error) {
	var wordfilters []Wordfilter
	err := db.Find(&wordfilters).Error
	return wordfilters, err
}
