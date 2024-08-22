package web

import (
	"errors"
	"strconv"
	"os"
	"time"
	"net/http"
	"IB1/db"
	"IB1/config"

	"github.com/gin-gonic/gin"
)

func redirect(f fErr, redirect string) func(*gin.Context) error {
	return func(c *gin.Context) error {
		err := f(c)
		if err != nil { return err }
		c.Redirect(http.StatusFound, redirect)
		return nil
	}
}

func needRank(c *gin.Context, rank int) error {
	ret := errors.New("insufficient privilege")
	var account db.Account
	account.Logged = false
	token := getCookie(c, "token")
	if token == "" { return ret }
	var err error
	account, err = db.GetAccountFromToken(token)
	if err != nil { return err }
	if account.Rank < rank { return ret }
	return nil
}

func handleConfig(f fErr, param string) func(c *gin.Context) {
	return catchCustom(redirect(hasRank(f, db.RANK_ADMIN), "/dashboard"),
				param, "/dashboard")
}

func canSetConfig(c *gin.Context, f fErr) fErr {
	if err := needRank(c, db.RANK_ADMIN); err != nil {
		return func(c *gin.Context) error { return err }
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

	indb, _ := c.GetPostForm("indb")
	config.Cfg.Media.InDatabase = indb == "on"

	title, ok := c.GetPostForm("title")
        if !ok { return errors.New("invalid form") }
	config.Cfg.Home.Title = title

	description, ok := c.GetPostForm("description")
        if !ok { return errors.New("invalid form") }
	config.Cfg.Home.Description = description

	domain, ok := c.GetPostForm("domain")
        if !ok { return errors.New("invalid form") }
	config.Cfg.Web.Domain = domain

	defaultname, ok := c.GetPostForm("defaultname")
        if !ok { return errors.New("invalid form") }
	config.Cfg.Post.DefaultName = defaultname

	tmp, ok := c.GetPostForm("tmp")
        if !ok { return errors.New("invalid form") }
	err := os.MkdirAll(config.Cfg.Media.Tmp, 0700)
	if err != nil { return err }
	config.Cfg.Media.Tmp = tmp

	path, _ := c.GetPostForm("media")
	if path == "" { path = config.Cfg.Media.Path }
	if !config.Cfg.Media.InDatabase {
		err = os.MkdirAll(path + "/thumbnail", 0700)
	}
	if err != nil { return err }
	config.Cfg.Media.Path = path

	sizeStr, _ := c.GetPostForm("maxsize")
	size, err := strconv.ParseUint(sizeStr, 10, 64)
	if err != nil { return err }
	config.Cfg.Media.MaxSize = size

	captcha, _ := c.GetPostForm("captcha")
	config.Cfg.Captcha.Enabled = captcha == "on"

	ascii, _ := c.GetPostForm("ascii")
	config.Cfg.Post.AsciiOnly = ascii == "on"

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

func createTheme(c *gin.Context) error {
	file, err := c.FormFile("theme")
        if err != nil { return err }
	name, hasName := c.GetPostForm("name")
        if !hasName { return errors.New("invalid form") }
	enabled, _ := c.GetPostForm("enabled")
	disabled := enabled != "on"
	data := make([]byte, file.Size)
	f, err := file.Open()
	if err != nil { return err }
	defer f.Close()
	_, err = f.Read(data)
	if err != nil { return err }
	data, err = minifyCSS(data)
	if err != nil { return err }
	err = db.AddTheme(name, string(data), disabled)
	if err != nil { return err }
	reloadThemes()
	return nil
}

func updateTheme(c *gin.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil { return errors.New("invalid theme") }
	name, hasName := c.GetPostForm("name")
        if !hasName { return errors.New("invalid form") }
	enabled, _ := c.GetPostForm("enabled")
	disabled := enabled != "on"
	err = db.UpdateThemeByID(id, name, disabled)
	if err != nil { return err }
	reloadThemes()
	return nil
}

func deleteTheme(c *gin.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil { return errors.New("invalid theme") }
	err = db.DeleteThemeByID(id)
	if err != nil { return err }
	reloadThemes()
	return nil
}

func addBan(c *gin.Context) error {
	ip, hasIP := c.GetPostForm("ip")
        if !hasIP { return errors.New("invalid form") }
	expiry, hasExpiry := c.GetPostForm("expiration")
	duration := int64(3600)
        if hasExpiry {
		expiration, err := time.Parse("2006-01-02T03:04", expiry)
		if err == nil {
			duration =  expiration.Unix() - time.Now().Unix()
		}
	}
	db.BanIP(ip, duration)
	return nil
}

func deleteBan(c *gin.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil { return errors.New("invalid theme") }
	err = db.RemoveBan(uint(id))
	return nil
}
