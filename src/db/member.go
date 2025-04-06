package db

import (
	"gorm.io/gorm"
	"encoding/json"
)

type MemberPrivilege int

func (p *MemberPrivilege) UnmarshalJSON(data []byte) error {
	var priv string
	err := json.Unmarshal(data, &priv)
	if err != nil { return err }
	v, ok := memberPrivileges[priv]
	if !ok {
		v = MemberPrivilege(NONE)
	}
	*p = v
	return nil
}

func (p MemberPrivilege) MarshalJSON() ([]byte, error) {
	return json.Marshal(Privilege(p).String())
}

func (p MemberPrivilege) Generic() Privilege {
	return Privilege(p)
}

var memberPrivileges = map[string]MemberPrivilege{
	"BAN_USER": 0,
	"APPROVE_MEDIA": 0,
	"REMOVE_MEDIA": 0,
	"REMOVE_POST": 0,
	"HIDE_POST": 0,
	"BYPASS_CAPTCHA": 0,
	"BYPASS_MEDIA_APPROVAL": 0,
	"VIEW_HIDDEN": 0,
	"VIEW_PENDING_MEDIA": 0,
	"VIEW_IP": 0,
	"BAN_IP": 0,
	"SHOW_RANK": 0,
	"CREATE_POST": 0,
	"CREATE_THREAD": 0,
	"PIN_THREAD": 0,
}

type MemberRank struct {
	gorm.Model
	CRUD[MemberRank]
	Name		string			`gorm:"unique"`
	Privileges	[]MemberPrivilege	`gorm:"serializer:json"`
}

type Membership struct {
	gorm.Model
	CRUD[Membership]
	MemberID	int	 `gorm:"uniqueIndex:idx_pair"`
	Member		Account
	BoardID		int	 `gorm:"uniqueIndex:idx_pair"`
	Board		Board
	RankID		int
	Rank		MemberRank
}

func GetMemberPrivilege(privilege string) MemberPrivilege {
	v, ok := memberPrivileges[privilege]
	if !ok { return MemberPrivilege(NONE) }
	return v
}

func GetMemberPrivileges() []string {
	v := []string{}
	for i := NONE + 1; i < LAST; i++ {
		_, ok := memberPrivileges[i.String()]
		if ok {
			v = append(v, i.String())
		}
	}
	return v
}

func GetMemberRanks() ([]MemberRank, error) {
	var ranks []MemberRank
	err := db.Find(&ranks).Error
	return ranks, err
}

func GetMemberRank(name string) (MemberRank, error) {
	var rank MemberRank
	err := db.Where("name = ?", name).Find(&rank).Error
	return rank, err
}

func (rank MemberRank) Has(privilege string) bool {
	priv := memberPrivileges[privilege]
	return rank.Can(priv)
}

func (rank MemberRank) Can(priv MemberPrivilege) bool {
	for _, v := range rank.Privileges {
		if v == priv { return true }
	}
	return false
}

func parseMemberPrivileges(privileges []string) []MemberPrivilege {
	privs := []MemberPrivilege{}
	for _, v := range privileges {
		priv := GetMemberPrivilege(v)
		if priv != MemberPrivilege(NONE) {
			privs = append(privs, priv)
		}
	}
	return privs
}

func CreateMemberRank(name string, privileges []string) error {
	return MemberRank{}.Add(MemberRank{
		Name: name, Privileges: parseMemberPrivileges(privileges),
	})
}

func UpdateMemberRank(id int, name string, privileges []string) error {
	v, err := unauthenticated.Get()
	if err != nil { return err }
	if v.ID == uint(id) {
		name = UNAUTHENTICATED
		unauthenticated.Refresh()
	}
	sessions.Clear()
	return MemberRank{}.Update(id, MemberRank{
		Name: name, Privileges: parseMemberPrivileges(privileges),
	})
}

func DeleteMemberRankByID(id int) error {
	return MemberRank{}.RemoveID(id, MemberRank{})
}

func ParseMemberPrivileges(privileges []string) []MemberPrivilege {
	privs := []MemberPrivilege{}
	for _, v := range privileges {
		priv := GetMemberPrivilege(v)
		if priv != MemberPrivilege(NONE) {
			privs = append(privs, priv)
		}
	}
	return privs
}
