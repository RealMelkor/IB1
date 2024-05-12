package web

import (
	"errors"
	"net/http"
	"IB1/db"
	"IB1/config"

	"github.com/gin-gonic/gin"
)

func handle(c *gin.Context, f func(*gin.Context) error, redirect string) {
	err := f(c)
	if err != nil {
		badRequest(c, err.Error())
		return
	}
	c.Redirect(http.StatusFound, redirect)
}

func isAdmin(c *gin.Context) bool {
	var account db.Account
	account.Logged = false
	token, _ := c.Cookie("session_token")
	if token == "" { return false }
	var err error
	account, err = db.GetAccountFromToken(token)
	if err != nil { return false }
	return account.Rank == db.RANK_ADMIN
}

func canSetConfig(c *gin.Context, f func(c *gin.Context) error) func(
						c *gin.Context) error {
	if !isAdmin(c) {
		return func(c *gin.Context) error {
			return errors.New("insufficient privilege")
		}
	}
	return f
}

func setDefaultTheme(c *gin.Context) error {
	theme, ok := c.GetPostForm("theme")
        if !ok { return errors.New("invalid form") }
        _, ok = themesTable[theme]
        if !ok { return errors.New("invalid theme") }
	config.Cfg.Home.Theme = theme
	return nil
}
