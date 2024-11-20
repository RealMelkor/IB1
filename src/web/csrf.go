package web

import (
        "errors"

        "github.com/labstack/echo/v4"
)

var tokens = map[string]string{}

func csrf(f echo.HandlerFunc) echo.HandlerFunc {
	invalid := errors.New("invalid csrf token")
	return func(c echo.Context) error {
		id, err := getID(c)
		if err != nil { return err }
		check := false
		value := ""
		if c.Request().Method == "POST" {
			value = c.Request().PostFormValue("csrf")
			check = true
		} else {
			for _, v := range c.ParamNames() {
				if v != "csrf" { continue }
				value = c.Param("csrf")
				check = true
				break
			}
		}
		if check {
			token, ok := tokens[id]
			if !ok { return invalid }
			if token != value { return invalid }
			tokens[id], err = newToken()
			if err != nil { return err }
		}
		_, ok := tokens[id]
		if !ok {
			tokens[id], err = newToken()
			if err != nil { return err }
		}
		return f(c)
	}
}
