package web

import (
        "errors"
	"IB1/util"

        "github.com/labstack/echo/v4"
)

func csrf(f echo.HandlerFunc) echo.HandlerFunc {
	invalid := errors.New("invalid csrf token")
	return func(c echo.Context) error {
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
			token := get(c)("csrf")
			if token != value { return invalid }
		}
		if check || get(c)("csrf") == nil {
			token, err := util.NewToken()
			if err != nil { return err }
			set(c)("csrf", token)
		}
		return f(c)
	}
}
