package db

import (
	"gorm.io/gorm"
	"encoding/json"
)

type Privilege int

func (p *Privilege) UnmarshalJSON(data []byte) error {
	var priv string
	err := json.Unmarshal(data, &priv)
	if err != nil { return err }
	*p = GetPrivilege(string(priv))
	return nil
}

func (p *Privilege) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.String())
}

//go:generate stringer -type=Privilege
const (
	NONE = Privilege(iota)
	CREATE_BOARD
	ADMINISTRATION
	MANAGE_USER
	BAN_USER
	APPROVE_MEDIA
	BAN_MEDIA
	REMOVE_MEDIA
	REMOVE_POST
	HIDE_POST
	BYPASS_CAPTCHA
	BYPASS_MEDIA_APPROVAL
	VIEW_HIDDEN
	VIEW_PENDING_MEDIA
	VIEW_IP
	BAN_IP
	SHOW_RANK
	LAST
)

type Rank struct {
	gorm.Model
	Name		string		`gorm:"unique"`
	Privileges	[]Privilege	`gorm:"serializer:json"`
}

var privileges = func() map[string]Privilege {
	m := make(map[string]Privilege)
	for i := NONE; i <= LAST; i++ {
		m[i.String()] = i
	}
	return m
}()

func GetPrivilege(privilege string) Privilege {
	v, ok := privileges[privilege]
	if !ok { return NONE }
	return v
}

func GetPrivileges() []string {
	v := []string{}
	for i := NONE + 1; i < LAST; i++ {
		v = append(v, i.String())
	}
	return v
}

func GetRanks() ([]Rank, error) {
	var ranks []Rank
	err := db.Find(&ranks).Error
	return ranks, err
}

func (rank Rank) Has(privilege string) bool {
	priv := privileges[privilege]
	for _, v := range rank.Privileges {
		if v == priv { return true }
	}
	return false
}

func parsePrivileges(privileges []string) []Privilege {
	privs := []Privilege{}
	for _, v := range privileges {
		priv := GetPrivilege(v)
		if priv != NONE {
			privs = append(privs, priv)
		}
	}
	return privs
}

func CreateRank(name string, privileges []string) error {
	return db.Create(&Rank{
		Name: name, Privileges: parsePrivileges(privileges)}).Error
}

func UpdateRank(id int, name string, privileges []string) error {
	return db.Where("id = ?", id).Updates(&Rank{
		Name: name, Privileges: parsePrivileges(privileges)}).Error
}

func DeleteRankByID(id int) error {
	return db.Unscoped().Delete(&Rank{}, id).Error
}
