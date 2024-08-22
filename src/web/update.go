package web

import (
	"errors"
	"strconv"
	"net/http"

	"github.com/gin-gonic/gin"

	"IB1/db"
)

func loginAs(c *gin.Context) error {
	name := c.PostForm("username")
	password := c.PostForm("password")
	err := verifyCaptcha(c)
	if err != nil { return err }
	token, err := db.Login(name, password)
	if err != nil { return errors.New("invalid credentials") }
	setCookie(c, "token", token)
	c.Redirect(http.StatusFound, "/")
	return nil
}

func disconnect(c *gin.Context) error {
	_, err := loggedAs(c)
	if err != nil { return err }
	db.Disconnect(getCookie(c, "token"))
	deleteCookie(c, "token")
	c.Redirect(http.StatusFound, "/")
	return nil
}

func remove(c *gin.Context) error {
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

func hide(c *gin.Context) error {
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

func ban(c *gin.Context) error {
	board := c.Param("board")
	ip := c.Param("ip")
	if err := db.BanIP(ip, 3600); err != nil { return err }
	c.Redirect(http.StatusFound, "/" + board)
	return nil
}

func newThread(c *gin.Context) error {

	if err := isBanned(c); err != nil { return err }

	boardName := c.Param("board")
	board, err := db.GetBoard(boardName)
	if err != nil { return err }

	name, hasName := c.GetPostForm("name")
	title, hasTitle := c.GetPostForm("title")
	content, hasContent := c.GetPostForm("content")
	if !hasTitle || !hasContent || !hasName || content == "" { 
		return errors.New("invalid form")
	}

	if err := checkCaptcha(c); err != nil { return err }

	media := ""
	file, err := c.FormFile("media")
	if err != nil { return err }
	if media, err = uploadFile(c, file); err != nil { return err }

	parsed, _ := parseContent(content, 0)
	number, err := db.CreateThread(board, title, name, media,
					c.ClientIP(), parsed)
	if err != nil { return err }

	c.Redirect(http.StatusFound, c.Request.URL.Path + "/" +
			strconv.Itoa(number))
	return nil
}

func newPost(c *gin.Context) error {

	if err := isBanned(c); err != nil { return err }

	boardName := c.Param("board")
	board, err := db.GetBoard(boardName)
	if err != nil { return err }

	threadNumberStr := c.Param("thread")
	threadNumber, err := strconv.Atoi(threadNumberStr)
	if err != nil { return err }
	thread, err := db.GetThread(board, threadNumber)
	if err != nil { return err }

	name, hasName := c.GetPostForm("name")
	content, hasContent := c.GetPostForm("content")
	if !hasName || !hasContent { return errors.New("invalid form") }

	if err := checkCaptcha(c); err != nil { return err }

	media := ""
	file, err := c.FormFile("media")
	if err == nil { 
		if media, err = uploadFile(c, file); err != nil { 
			return err
		}
	}

	parsed, refs := parseContent(content, thread.ID)
	number, err := db.CreatePost(thread, parsed, name, media,
					c.ClientIP(), nil)
	if err != nil { return err }

	for _, v := range refs {
		db.CreateReference(thread.ID, number, v)
	}

	c.Redirect(http.StatusFound, c.Request.URL.Path)
	return nil
}
