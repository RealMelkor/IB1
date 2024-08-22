package web

import (
	"errors"
	"github.com/gin-gonic/gin"

	"IB1/config"
	"IB1/db"
)

type session map[string]any
var sessions = map[string]session{}

func setCookie(c *gin.Context, name string, value string) {
	c.SetCookie(name, value, 0, "/", config.Cfg.Web.Domain, false, true)
}

func getCookie(c *gin.Context, name string) string {
	v, err := c.Cookie(name)
	if err != nil { return "" }
	return v
}

func deleteCookie(c *gin.Context, name string) {
	c.SetCookie(name, "", 1, "/", config.Cfg.Web.Domain, false, true)
}

func getID(c *gin.Context) (string, error) {
	v := getCookie(c, "id")
	if v == "" {
		token, err := newToken()
		if err != nil { return "", err }
		setCookie(c, "id", token)
	}
	_, ok := sessions[v]
	if !ok { sessions[v] = session{} }
	return v, nil
}

func get(c *gin.Context) func(string)any {
	id, err := getID(c)
	if err != nil { return func(string)any { return nil } }
	return func(param string)any {
		v, ok := sessions[id][param]
		if !ok { return nil }
		return v
	}
}

func set(c *gin.Context) func(string, any) any {
	id, err := getID(c)
	if err != nil { return func(string, any) any { return "" } }
	return func(param string, value any) any {
		_, ok := sessions[id]
		if !ok { sessions[id] = session{} }
		sessions[id][param] = value
		return nil
	}
}

func once(c *gin.Context) func(string)any {
	id, err := getID(c)
	if err != nil { return func(string)any { return nil } }
	return func(param string)any {
		v, ok := sessions[id][param]
		if !ok { return nil }
		delete(sessions[id], param)
		return v
	}
}

func has(c *gin.Context) func(string)bool {
	id, err := getID(c)
	if err != nil { return func(string)bool { return false } }
	return func(param string)bool {
		_, ok := sessions[id][param]
		return ok
	}
}

func loggedAs(c *gin.Context) (db.Account, error) {
	token := getCookie(c, "token")
	if token == "" { return db.Account{}, errors.New("unauthenticated") }
	return db.GetAccountFromToken(token)
}
