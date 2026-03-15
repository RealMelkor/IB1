package web

import (
	"IB1/db"
)

func createMemberRank(name string, privileges []string) error {
	return db.MemberRank{}.Add(db.MemberRank{
		Name:       name,
		Privileges: db.ParseMemberPrivileges(privileges),
	})
}

func updateMemberRank(id int, name string, privileges []string) error {
	return db.MemberRank{}.Update(id, db.MemberRank{
		Name:       name,
		Privileges: db.ParseMemberPrivileges(privileges),
	})
}

func deleteMemberRank(id int) error {
	return db.MemberRank{}.RemoveID(id, db.MemberRank{})
}
