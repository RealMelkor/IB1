package web

import (
	"github.com/dchest/captcha"
	"github.com/gin-gonic/gin"
	"net/http"
	"IB1/config"
)

func captchaNew(c *gin.Context) error {
	if !config.Cfg.Captcha.Enabled { return nil }
	id, err := getID(c)
	if err != nil { return err }
	captchaID, ok := sessions[id]["captcha"]
	if !ok {
		sessions[id]["captcha"] = captcha.New()
	} else {
		captcha.Reload(captchaID.(string))
	}
	return nil
}

func captchaImage(c *gin.Context) {
	if err := captchaNew(c); err != nil {
		badRequest(c, err.Error())
		return
	}
	id := get(c)("captcha").(string)
	if id == "" {
		badRequest(c, "no captcha")
		return
	}
	c.Status(http.StatusOK)
	captcha.WriteImage(c.Writer, id, captcha.StdWidth, captcha.StdHeight)
}

func captchaVerify(c *gin.Context, answer string) bool {
	cookie, err := c.Cookie("captcha")
	if err != nil || cookie == "" { return false }
	return captcha.VerifyString(cookie, answer)
}
