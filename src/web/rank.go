package web

import (
	"github.com/labstack/echo/v4"
	"strconv"

	"IB1/db"
)

func parsePrivileges(c echo.Context) []string {
	privileges := []string{}
	for _, v := range db.GetPrivileges() {
		priv, _ := getPostForm(c, v)
		if priv == "on" {
			privileges = append(privileges, v)
		}
	}
	return privileges
}

func createRank(c echo.Context) error {
	name, hasName := getPostForm(c, "name")
        if !hasName { return errInvalidForm }
	return db.CreateRank(name, parsePrivileges(c))
}

func updateRank(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil { return err }
	name, hasName := getPostForm(c, "name")
        if !hasName { return errInvalidForm }
	return db.UpdateRank(id, name, parsePrivileges(c))
}

func deleteRank(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil { return err }
	return db.DeleteRankByID(id)
}

func createMemberRank(c echo.Context) error {
        name, hasName := getPostForm(c, "name")
        if !hasName { return errInvalidForm }
	return db.MemberRank{}.Add(db.MemberRank{
                Name: name,
		Privileges: db.ParseMemberPrivileges(parsePrivileges(c)),
        })
}

func updateMemberRank(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil { return err }
	name, hasName := getPostForm(c, "name")
        if !hasName { return errInvalidForm }
	return db.MemberRank{}.Update(id, db.MemberRank{
                Name: name,
		Privileges: db.ParseMemberPrivileges(parsePrivileges(c)),
        })
}

func deleteMemberRank(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil { return err }
	return db.MemberRank{}.RemoveID(id, db.MemberRank{})
}
