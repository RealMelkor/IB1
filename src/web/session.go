package web

import (
	"errors"
	"github.com/labstack/echo/v4"
	"net/http"
	"time"

	"IB1/config"
	"IB1/db"
	"IB1/util"
)

const KeyValueLifespan = time.Hour * 48

type KeyValue struct {
	Value    any
	Creation time.Time
}

var sessions = util.SafeMap[db.SessionKey, db.KeyValue]{}

func clearSession() {
	for {
		time.Sleep(KeyValueLifespan)
		sessions.Iter(func(key db.SessionKey, v db.KeyValue) (db.KeyValue, bool) {
			diff := time.Since(v.Creation)
			return v, diff > KeyValueLifespan
		})
		db.SaveSessions(&sessions)
	}
}

func setCookie(c echo.Context, name string, value string) {
	cookie := http.Cookie{
		Path:   "/",
		Domain: config.Cfg.Web.Domain,
		Name:   name,
		Value:  value,
		Secure: true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	c.SetCookie(&cookie)
}

func setCookiePermanent(c echo.Context, name string, value string) {
	cookie := http.Cookie{
		Path:    "/",
		Domain:  config.Cfg.Web.Domain,
		Name:    name,
		Value:   value,
		Secure:	 true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Expires: time.Now().Add(3600 * 24 * time.Hour),
	}
	c.SetCookie(&cookie)
}

func getCookie(c echo.Context, name string) string {
	v, err := c.Cookie(name)
	if err != nil {
		return ""
	}
	return v.Value
}

func deleteCookie(c echo.Context, name string) {
	cookie := http.Cookie{
		Domain:  config.Cfg.Web.Domain,
		Name:    name,
		Secure:	 true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
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
			if err != nil {
				return "", err
			}
			setCookie(c, "id", token)
			v = token
		}
	}
	return v, nil
}

func get(c echo.Context) func(string) any {
	id, err := getID(c)
	if err != nil {
		return func(string) any { return nil }
	}
	return func(param string) any {
		v, ok := sessions.Get(db.GetSessionKey(id, param))
		if !ok {
			return nil
		}
		return v.Value
	}
}

func set(c echo.Context) func(string, any) any {
	id, err := getID(c)
	if err != nil {
		return func(string, any) any { return "" }
	}
	return func(param string, value any) any {
		v := db.KeyValue{
			Creation: time.Now(), Value: value, Key: param,
		}
		sessions.Set(db.GetSessionKey(id, param), v)
		return nil
	}
}

func once(c echo.Context) func(string) any {
	id, err := getID(c)
	if err != nil {
		return func(string) any { return nil }
	}
	return func(param string) any {
		v, ok := sessions.Get(db.GetSessionKey(id, param))
		if !ok {
			return nil
		}
		sessions.Delete(db.GetSessionKey(id, param))
		return v.Value
	}
}

func has(c echo.Context) func(string) bool {
	id, err := getID(c)
	if err != nil {
		return func(string) bool { return false }
	}
	return func(param string) bool {
		_, ok := sessions.Get(db.GetSessionKey(id, param))
		return ok
	}
}

func loggedAs(c echo.Context) (db.Account, error) {
	token := getCookie(c, "token")
	if token == "" {
		return db.Account{}, errors.New("unauthenticated")
	}
	return db.GetAccountFromToken(token)
}
