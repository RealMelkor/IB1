package web

import (
	"errors"
	"strconv"
	"net/http"

	"github.com/labstack/echo/v4"

	"IB1/db"
	"IB1/config"
)

func readOnly(f echo.HandlerFunc) echo.HandlerFunc {
	if !config.Cfg.Post.ReadOnly { return f }
	return func(echo.Context) error {
		return errors.New("The website is currently read-only")
	}
}

func loginAs(c echo.Context) error {
	name := c.Request().PostFormValue("username")
	password := c.Request().PostFormValue("password")
	err := verifyCaptcha(c)
	if err != nil { return err }
	token, err := db.Login(name, password)
	if err != nil { return errors.New("invalid credentials") }
	setCookie(c, "token", token)
	theme, err := db.GetUserTheme(name)
	if err != nil { return err }
	setCookiePermanent(c, "theme", theme)
	c.Redirect(http.StatusFound, "/")
	return nil
}

func register(c echo.Context) error {
	name := c.Request().PostFormValue("username")
	password := c.Request().PostFormValue("password")
	confirm := c.Request().PostFormValue("confirm")
	if confirm != password { return errors.New("passwords don't match") }
	err := verifyCaptcha(c)
	if err != nil { return err }
	err = db.CreateAccount(name, password, db.RANK_USER)
	if err != nil { return err }
	token, err := db.Login(name, password)
	if err != nil { return errors.New("invalid credentials") }
	setCookie(c, "token", token)
	c.Redirect(http.StatusFound, "/")
	return nil
}

func disconnect(c echo.Context) error {
	_, err := loggedAs(c)
	if err != nil { return err }
	db.Disconnect(getCookie(c, "token"))
	deleteCookie(c, "token")
	c.Redirect(http.StatusFound, "/")
	return nil
}

func remove(c echo.Context) error {
	board := c.Param("board")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil { return err }

	post, err := db.GetPostFromBoard(board, id)
	if err != nil { return err }
	err = db.Remove(board, id)
	if err != nil { return err }

	dst := "/" + board
	if id != post.Thread.Number {
		dst += "/" + strconv.Itoa(post.Thread.Number)
	}
	c.Redirect(http.StatusFound, dst)
	return nil
}

func cancel(c echo.Context) error {
	board := c.Param("board")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil { return err }

	post, err := db.GetPostFromBoard(board, id)
	if err != nil { return err }
	if post.Session != getCookie(c, "id") || post.Session == "" {
		user, err := loggedAs(c)
		if err != nil || user.ID != post.OwnerID {
			return errors.New("invalid post")
		}
	}

	err = db.Remove(board, id)
	if err != nil { return err }

	dst := "/" + board
	if id != post.Thread.Number {
		dst += "/" + strconv.Itoa(post.Thread.Number)
	}
	c.Redirect(http.StatusFound, dst)
	return nil
}

func hide(c echo.Context) error {
	var id int
	var post db.Post
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil { return err }
	board := c.Param("board")
	post, err = db.GetPostFromBoard(board, id)
	if err != nil { return err }
	err = db.Hide(board, id, post.Disabled)
	if err != nil { return err }
	c.Redirect(http.StatusFound, "/" + board + "/" +
			strconv.Itoa(post.Thread.Number))
	return err
}

func ban(c echo.Context) error {
	board := c.Param("board")
	ip := c.Param("ip")
	if err := db.BanIP(ip, 3600); err != nil { return err }
	c.Redirect(http.StatusFound, "/" + board)
	return nil
}

func newThread(c echo.Context) error {

	if err := isBanned(c); err != nil { return err }

	boardName := c.Param("board")
	board, err := db.GetBoard(boardName)
	if err != nil { return err }

	name, _ := getPostForm(c, "name")
	title, _ := getPostForm(c, "title")
	signed, _ := getPostForm(c, "signed")
	rank, _ := getPostForm(c, "rank")
	content, hasContent := getPostForm(c, "content")
	if !hasContent || content == "" {
		return errors.New("invalid form")
	}

	if err := checkCaptcha(c); err != nil { return err }

	media := ""
	file, err := c.FormFile("media")
	if err != nil { return err }
	user, err := loggedAs(c)
	if err == nil && signed == "on" { name = user.Name }
	if media, err = uploadFile(file, user.HasRank("trusted")); err != nil {
		return err
	}

	parsed, _ := parseContent(content, 0)
	number, err := db.CreateThread(board, title, name, media, clientIP(c),
					getCookie(c, "id"), user,
					signed == "on", rank == "on", parsed)
	if err != nil { return err }

	c.Redirect(http.StatusFound, c.Request().URL.Path + "/" +
			strconv.Itoa(number))
	return nil
}

func newPost(c echo.Context) error {

	if err := isBanned(c); err != nil { return err }

	boardName := c.Param("board")
	board, err := db.GetBoard(boardName)
	if err != nil { return err }

	threadNumberStr := c.Param("thread")
	threadNumber, err := strconv.Atoi(threadNumberStr)
	if err != nil { return err }
	thread, err := db.GetThread(board, threadNumber)
	if err != nil { return err }

	name, _ := getPostForm(c, "name")
	content, _ := getPostForm(c, "content")
	signed, _ := getPostForm(c, "signed")
	rank, _ := getPostForm(c, "rank")

	if err := checkCaptcha(c); err != nil { return err }

	media := ""
	user, err := loggedAs(c)
	if err == nil && signed == "on" { name = user.Name }
	file, err := c.FormFile("media")
	if err == nil { 
		media, err = uploadFile(file, user.HasRank("trusted"))
		if err != nil { return err }
	}

	parsed, refs := parseContent(content, thread.ID)
	number, err := db.CreatePost(thread, parsed, name, media, clientIP(c),
			getCookie(c, "id"), user, signed == "on", rank == "on",
			nil)
	if err != nil { return err }

	for _, v := range refs {
		db.CreateReference(thread.ID, number, v)
	}

	c.Redirect(http.StatusFound, c.Request().URL.Path)
	return nil
}
