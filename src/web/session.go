package web

import (
	"errors"
	"time"
	"net/http"
	"github.com/labstack/echo/v4"

	"IB1/config"
	"IB1/db"
	"IB1/util"
)

const KeyValueLifespan = time.Hour * 48

type KeyValue struct {
	Value		any
	Creation	time.Time
}

var sessions = util.SafeMap[db.KeyValues]{}

func clearSession() {
	for {
		time.Sleep(KeyValueLifespan)
		sessions.Iter(func(key string, s db.KeyValues)(db.KeyValues, bool){
			for k, v := range s {
				diff := time.Now().Sub(v.Creation)
				if diff > KeyValueLifespan {
					delete(s, k)
				}
			}
			return s, len(s) > 0
		})
		db.SaveSessions(&sessions)
	}
}

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
		Expires: time.Now().Add(3600 * 24 * time.Hour),
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
		raw := c.Response().Header().Get("Set-Cookie")
		if raw != "" {
			header := http.Header{}
			header.Add("Cookie", raw)
			request := http.Request{Header: header}
			cookie, err := request.Cookie("id")
			if err == nil {
				v = cookie.Value
			}
		}
		if v == "" {
			token, err := util.NewToken()
			if err != nil { return "", err }
			setCookie(c, "id", token)
			v = token
		}
	}
	_, ok := sessions.Get(v)
	if !ok { sessions.Set(v, db.KeyValues{}) }
	return v, nil
}

func get(c echo.Context) func(string)any {
	id, err := getID(c)
	if err != nil { return func(string)any { return nil } }
	return func(param string)any {
		m, ok := sessions.Get(id)
		if !ok { return nil }
		v, ok := m[param]
		if !ok { return nil }
		return v.Value
	}
}

func set(c echo.Context) func(string, any) any {
	id, err := getID(c)
	if err != nil { return func(string, any) any { return "" } }
	return func(param string, value any) any {
		_, ok := sessions.Get(id)
		if !ok { sessions.Set(id, db.KeyValues{}) }
		m, ok := sessions.Get(id)
		if !ok { return nil }
		m[param] = db.KeyValue{
			Creation: time.Now(), Value: value, Key: param,
		}
		sessions.Set(id, m)
		return nil
	}
}

func once(c echo.Context) func(string)any {
	id, err := getID(c)
	if err != nil { return func(string)any { return nil } }
	return func(param string)any {
		m, ok := sessions.Get(id)
		if !ok { return nil }
		v, ok := m[param]
		delete(m, param)
		sessions.Set(id, m)
		return v.Value
	}
}

func has(c echo.Context) func(string)bool {
	id, err := getID(c)
	if err != nil { return func(string)bool { return false } }
	return func(param string)bool {
		m, ok := sessions.Get(id)
		if !ok { return false }
		_, ok = m[param]
		return ok
	}
}

func loggedAs(c echo.Context) (db.Account, error) {
	token := getCookie(c, "token")
	if token == "" { return db.Account{}, errors.New("unauthenticated") }
	return db.GetAccountFromToken(token)
}
