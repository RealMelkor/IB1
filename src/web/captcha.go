package web

import (
	"errors"
	"net/http"

	"github.com/dchest/captcha"
	"github.com/gin-gonic/gin"

	"IB1/config"
)

func captchaNew(c *gin.Context) (string, error) {
	id, err := getID(c)
	if err != nil { return "", err }
	captchaID := captcha.New()
	sessions[id]["captcha"] = captchaID
	return captchaID, nil
}

func captchaImage(c *gin.Context) {
	id, err := captchaNew(c)
	if err != nil {
		badRequest(c, err.Error())
		return
	}
	c.Status(http.StatusOK)
	captcha.WriteImage(c.Writer, id, captcha.StdWidth, captcha.StdHeight)
}

func captchaVerify(c *gin.Context, answer string) bool {
	v := get(c)("captcha").(string)
	if v == "" { return false }
	return captcha.VerifyString(v, answer)
}

func checkCaptcha(c *gin.Context) error {
	if !config.Cfg.Captcha.Enabled { return nil }
	_, err := loggedAs(c)
	if err == nil { return nil } // captcha not needed if logged
	return verifyCaptcha(c)
}

func verifyCaptcha(c *gin.Context) error {
	if !config.Cfg.Captcha.Enabled { return nil }
	captcha, hasCaptcha := c.GetPostForm("captcha")
	if !hasCaptcha { return errors.New("invalid form") }
	if !captchaVerify(c, captcha) { return errors.New("wrong captcha") }
	return nil
}
