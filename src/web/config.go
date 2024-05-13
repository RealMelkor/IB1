package web

import (
	"errors"
	"strconv"
	"os"
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
        _, ok = getThemesTable()[theme]
        if !ok { return errors.New("invalid theme") }
	config.Cfg.Home.Theme = theme
	return nil
}

func updateConfig(c *gin.Context) error {
	if err := setDefaultTheme(c); err != nil { return err }

	title, ok := c.GetPostForm("title")
        if !ok { return errors.New("invalid form") }
	config.Cfg.Home.Title = title

	description, ok := c.GetPostForm("description")
        if !ok { return errors.New("invalid form") }
	config.Cfg.Home.Description = description

	domain, ok := c.GetPostForm("domain")
        if !ok { return errors.New("invalid form") }
	config.Cfg.Web.Domain = domain

	tmp, ok := c.GetPostForm("tmp")
        if !ok { return errors.New("invalid form") }
	err := os.MkdirAll(config.Cfg.Media.Tmp, 0700)
	if err != nil { return err }
	config.Cfg.Media.Tmp = tmp

	captcha, _ := c.GetPostForm("captcha")
	config.Cfg.Captcha.Enabled = captcha == "on"

	path, _ := c.GetPostForm("media")
	err = os.MkdirAll(path + "/thumbnail", 0700)
	if err != nil { return err }
	config.Cfg.Media.Path = path

	return db.UpdateConfig()
}

func createBoard(c *gin.Context) error {
	board, hasBoard := c.GetPostForm("board")
	name, hasName := c.GetPostForm("name")
        if !hasBoard || !hasName { return errors.New("invalid form") }
	description, _ := c.GetPostForm("description")
	err := db.CreateBoard(board, name, description)
	if err != nil { return err }
	return db.LoadBoards()
}

func updateBoard(c *gin.Context) error {
	board, hasBoard := c.GetPostForm("board")
	name, hasName := c.GetPostForm("name")
        if !hasBoard || !hasName { return errors.New("invalid form") }
	enabled, _ := c.GetPostForm("enabled")
	description, _ := c.GetPostForm("description")
	boards, err := db.GetBoards()
	if err != nil { return err }
	for _, v := range boards {
		if strconv.Itoa(int(v.ID)) != c.Param("board") { continue }
		v.Name = board
		v.LongName = name
		v.Description = description
		v.Disabled = enabled != "on"
		if err := db.UpdateBoard(v); err != nil { return err }
		return db.LoadBoards()
	}
        return errors.New("invalid board")
}

func deleteBoard(c *gin.Context) error {
	for i, v := range db.Boards {
		if strconv.Itoa(int(v.ID)) != c.Param("board") { continue }
		err := db.DeleteBoard(v)
		if err != nil { return err }
		delete(db.Boards, i)
		return nil
	}
        return errors.New("invalid board")
}

func handleConfig(f func(c *gin.Context) error) func(c *gin.Context) {
	return func(c *gin.Context) {
		handle(c, canSetConfig(c, f), "/dashboard")
	}
}
