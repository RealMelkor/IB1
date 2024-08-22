package web

import (
	"errors"
	"time"
	"net/http"
	"github.com/labstack/echo/v4"

	"IB1/config"
	"IB1/db"
)

type session map[string]any
var sessions = map[string]session{}

func setCookie(c echo.Context, name string, value string) {
	cookie := http.Cookie{
		Path: "/",
                Domain: config.Cfg.Web.Domain,
                Name: name,
                Value: value,
        }
	c.SetCookie(&cookie)
}

func setCookiePermanent(c echo.Context, name string, value string) {
	cookie := http.Cookie{
		Path: "/",
                Domain: config.Cfg.Web.Domain,
                Name: name,
                Value: value,
		Expires: time.Now().Add(3650 * 24 * time.Hour),
        }
	c.SetCookie(&cookie)
}

func getCookie(c echo.Context, name string) string {
	v, err := c.Cookie(name)
	if err != nil { return "" }
	return v.Value
}

func deleteCookie(c echo.Context, name string) {
	cookie := http.Cookie{
                Domain: config.Cfg.Web.Domain,
                Name: name,
		Expires: time.Now().UTC().Add(time.Duration(-86400)),
        }
	c.SetCookie(&cookie)
}

func getID(c echo.Context) (string, error) {
	v := getCookie(c, "id")
	if v == "" {
		token, err := newToken()
		if err != nil { return "", err }
		setCookie(c, "id", token)
		v = token
	}
	_, ok := sessions[v]
	if !ok { sessions[v] = session{} }
	return v, nil
}

func get(c echo.Context) func(string)any {
	id, err := getID(c)
	if err != nil { return func(string)any { return nil } }
	return func(param string)any {
		v, ok := sessions[id][param]
		if !ok { return nil }
		return v
	}
}

func set(c echo.Context) func(string, any) any {
	id, err := getID(c)
	if err != nil { return func(string, any) any { return "" } }
	return func(param string, value any) any {
		_, ok := sessions[id]
		if !ok { sessions[id] = session{} }
		sessions[id][param] = value
		return nil
	}
}

func once(c echo.Context) func(string)any {
	id, err := getID(c)
	if err != nil { return func(string)any { return nil } }
	return func(param string)any {
		v, ok := sessions[id][param]
		if !ok { return nil }
		delete(sessions[id], param)
		return v
	}
}

func has(c echo.Context) func(string)bool {
	id, err := getID(c)
	if err != nil { return func(string)bool { return false } }
	return func(param string)bool {
		_, ok := sessions[id][param]
		return ok
	}
}

func loggedAs(c echo.Context) (db.Account, error) {
	token := getCookie(c, "token")
	if token == "" { return db.Account{}, errors.New("unauthenticated") }
	return db.GetAccountFromToken(token)
}
