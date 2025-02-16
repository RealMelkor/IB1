package db

import (
	"gorm.io/gorm"
	"encoding/json"
	"errors"

	"IB1/util"
)

const (
	UNAUTHENTICATED = "unauthenticated"
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
	BYPASS_READONLY
	VIEW_PRIVATE
	LAST
)

type Rank struct {
	gorm.Model
	CRUD[Rank]
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
	return rank.Can(priv)
}

func (rank Rank) Can(priv Privilege) bool {
	for _, v := range rank.Privileges {
		if v == priv { return true }
	}
	return false
}

var unauthenticated = util.SafeObj[Rank]{
	Reload: func() (Rank, error) {
		return GetRank(UNAUTHENTICATED)
	},
}

func UnauthenticatedCan(privilege string) (bool, error) {
	return AsUnauthenticated(GetPrivilege(privilege))
}

func AsUnauthenticated(privilege Privilege) (bool, error) {
	v, err := unauthenticated.Get()
	if err != nil { return false, err }
	return v.Can(privilege), nil
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
	return Rank{}.Add(Rank{
		Name: name, Privileges: parsePrivileges(privileges),
	})
}

func UpdateRank(id int, name string, privileges []string) error {
	v, err := unauthenticated.Get()
	if err != nil { return err }
	if v.ID == uint(id) {
		name = UNAUTHENTICATED
		unauthenticated.Refresh()
	}
	sessions.Clear()
	return Rank{}.Update(id, Rank{
		Name: name, Privileges: parsePrivileges(privileges),
	})
}

func DeleteRankByID(id int) error {
	v, err := unauthenticated.Get()
	if err != nil { return err }
	if v.ID == uint(id) {
		return errors.New("Cannot delete 'unauthenticated' group")
	}
	sessions.Clear()
	return Rank{}.RemoveID(id, Rank{})
}
