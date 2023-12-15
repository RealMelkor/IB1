package web

import (
	"github.com/dchest/captcha"
	"github.com/gin-gonic/gin"
	"net/http"
	"IB1/config"
)

func captchaNew(c *gin.Context) {
	if !config.Cfg.Captcha.Enabled { return }
	c.SetCookie("captcha", captcha.NewLen(config.Cfg.Captcha.Length),
		int(captcha.Expiration.Seconds()), "/",
		config.Cfg.Web.Domain, true, false)
}

func captchaImage(c *gin.Context) {
	cookie, err := c.Cookie("captcha")
	if err != nil || cookie == "" {
		badRequest(c, "no captcha")
		return
	}
	c.Status(http.StatusOK)
	captcha.WriteImage(c.Writer, cookie,
			captcha.StdWidth, captcha.StdHeight)
}

func captchaVerify(c *gin.Context, answer string) bool {
	cookie, err := c.Cookie("captcha")
	if err != nil || cookie == "" { return false }
	return captcha.VerifyString(cookie, answer)
}
