package db

import (
	"gorm.io/gorm"
)

type Theme struct {
	gorm.Model
	Content		string
	Name		string `gorm:"unique"`
	Disabled	bool
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
