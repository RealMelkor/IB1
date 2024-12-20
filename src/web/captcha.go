package web

import (
	"errors"
	"net/http"

	"github.com/dchest/captcha"
	"github.com/labstack/echo/v4"

	"IB1/config"
	"IB1/db"
)

func captchaNew(c echo.Context) (string, error) {
	captchaID := captcha.New()
	set(c)("captcha", captchaID)
	return captchaID, nil
}

func captchaImage(c echo.Context) error {
	id, err := captchaNew(c)
	if err != nil {
		return badRequest(c, err)
	}
	c.Response().WriteHeader(http.StatusOK)
	return captcha.WriteImage(c.Response().Writer, id,
			captcha.StdWidth, captcha.StdHeight)
}

func captchaVerify(c echo.Context, answer string) bool {
	v := get(c)("captcha")
	if v == nil { return false }
	return captcha.VerifyString(v.(string), answer)
}

func checkCaptcha(c echo.Context) error {
	if !config.Cfg.Captcha.Enabled { return nil }
	// trusted users don't need captcha
	if user, err := loggedAs(c); err == nil {
		if err := user.Can(db.BYPASS_CAPTCHA); err == nil {
			return nil
		}
	}
	return verifyCaptcha(c)
}

func verifyCaptcha(c echo.Context) error {
	if !config.Cfg.Captcha.Enabled { return nil }
	captcha, hasCaptcha := getPostForm(c, "captcha")
	if !hasCaptcha { return errors.New("invalid form") }
	if !captchaVerify(c, captcha) { return errors.New("wrong captcha") }
	return nil
}
