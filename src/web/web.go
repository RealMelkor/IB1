package web

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"log"
	"os"
	"errors"
	
	"IB1/db"
	"IB1/config"
)

func render(template string, data any, c *gin.Context) error {
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Header().Add("Content-Type", "text/html; charset=utf-8")
	err := templates.Lookup(template).Execute(c.Writer, data)
	if err != nil { return err }
	return nil
}

func internalError(c *gin.Context, data string) {
	c.Data(http.StatusBadRequest, "text/plain", []byte(data))
}

func badRequest(c *gin.Context, data string) {
	log.Println(data)
	c.Data(http.StatusBadRequest, "text/plain", []byte("bad request"))
}

func badRequestExplicit(c *gin.Context, data string) {
	c.Data(http.StatusBadRequest, "text/plain", []byte(data))
}

func index(c *gin.Context) {
	if err := renderIndex(c); err != nil {
		internalError(c, err.Error())
		return
	}
}

func boardIndex(c *gin.Context) {
	boardName := c.Param("board")
	board, err := db.GetBoard(boardName)
	if err != nil { 
		internalError(c, err.Error())
		return
	}
	if len(board.Threads) > 4 {
		// TODO: support pages
		board.Threads = board.Threads[0:4]
	}
	for i := range board.Threads {
		if err := db.RefreshThread(&board.Threads[i]); err != nil {
			internalError(c, err.Error())
			return
		}
		if length := len(board.Threads[i].Posts); length > 5 {
			posts := []db.Post{board.Threads[i].Posts[0]}
			board.Threads[i].Posts = append(posts,
				board.Threads[i].Posts[length - 4:]...)
		}
	}
	captchaNew(c)
	if err := renderBoard(board, c); err != nil {
		internalError(c, err.Error())
		return
	}
}

func catalog(c *gin.Context) {
	boardName := c.Param("board")
	board, err := db.GetBoard(boardName)
	if err != nil { 
		internalError(c, err.Error())
		return
	}
	for i, v := range board.Threads {
		err := db.RefreshThread(&v)
		if err != nil {
			internalError(c, err.Error())
			return
		}
		v.Replies = len(v.Posts) - 1
		v.Images = -1
		for _, post := range v.Posts {
			if post.Media != "" {
				v.Images++
			}
		}
		board.Threads[i] = v
	}
	captchaNew(c)
	if err := renderCatalog(board, c); err != nil {
		internalError(c, err.Error())
		return
	}
}

func checkCaptcha(c *gin.Context) bool {
	if config.Cfg.Captcha.Enabled {
		_, err := loggedAs(c)
		if err == nil { return true }
		captcha, hasCaptcha := c.GetPostForm("captcha")
		if !hasCaptcha {
			badRequestExplicit(c, "invalid form")
			return false
		}
		if !captchaVerify(c, captcha) {
			badRequestExplicit(c, "wrong captcha")
			return false
		}
	}
	return true
}

func verifyCaptcha(c *gin.Context) error {
	if !config.Cfg.Captcha.Enabled { return nil }
	captcha, hasCaptcha := c.GetPostForm("captcha")
	if !hasCaptcha {
		return errors.New("invalid form")
	}
	if !captchaVerify(c, captcha) {
		return errors.New("wrong captcha")
	}
	return nil
}

func newThread(c *gin.Context) {

	boardName := c.Param("board")
	board, err := db.GetBoard(boardName)
	if err != nil { 
		internalError(c, err.Error())
		return 
	}

	name, hasName := c.GetPostForm("name")
	title, hasTitle := c.GetPostForm("title")
	content, hasContent := c.GetPostForm("content")
	if !hasTitle || !hasContent || !hasName || content == "" { 
		badRequestExplicit(c, "invalid form")
		return 
	}

	if !checkCaptcha(c) { return }

	media := ""
	file, err := c.FormFile("media")
	if err != nil { 
		badRequest(c, err.Error())
		return 
	}
	if media, err = uploadFile(c, file); err != nil { 
		badRequest(c, err.Error())
		return
	}

	parsed, _ := parseContent(content, 0)
	number, err := db.CreateThread(board, title, name, media,
					c.ClientIP(), parsed)
	if err != nil { 
		internalError(c, err.Error())
		return
	}

	c.Redirect(http.StatusFound, c.Request.URL.Path + "/" +
			strconv.Itoa(number))
}

func newPost(c *gin.Context) {

	boardName := c.Param("board")
	board, err := db.GetBoard(boardName)
	if err != nil { 
		badRequest(c, err.Error())
		return
	}

	threadNumberStr := c.Param("thread")
	threadNumber, err := strconv.Atoi(threadNumberStr)
	if err != nil { 
		badRequest(c, err.Error())
		return
	}
	thread, err := db.GetThread(board, threadNumber)
	if err != nil {
		badRequest(c, err.Error())
		return
	}

	name, hasName := c.GetPostForm("name")
	content, hasContent := c.GetPostForm("content")
	if !hasName || !hasContent {
		badRequest(c, "invalid form")
		return
	}

	if !checkCaptcha(c) { return }

	media := ""
	file, err := c.FormFile("media")
	if err == nil { 
		if media, err = uploadFile(c, file); err != nil { 
			badRequest(c, err.Error())
			return
		}
	}

	parsed, refs := parseContent(content, thread.ID)
	number, err := db.CreatePost(thread, parsed, name, media,
					c.ClientIP(), nil)
	if err != nil {
		internalError(c, err.Error())
		return
	}

	for _, v := range refs {
		db.CreateReference(thread.ID, number, v)
	}

	c.Redirect(http.StatusFound, c.Request.URL.Path)
}

func thread(c *gin.Context) {

	var thread db.Thread
	var board db.Board

	threadID := c.Param("thread")
	boardName := c.Param("board")

	id, err := strconv.Atoi(threadID)
	if err != nil {
		badRequest(c, err.Error())
		return
	}
	board, err = db.GetBoard(boardName)
	if err != nil {
		internalError(c, err.Error())
		return
	}
	thread, err = db.GetThread(board, id)
	if err != nil {
		internalError(c, err.Error())
		return
	}
	if thread.Posts[0].Disabled {
		if _, err := loggedAs(c); err != nil {
			badRequest(c, "not found")
			return
		}
	}
	captchaNew(c)
	if err := renderThread(thread, c); err != nil {
		internalError(c, err.Error())
		return
	}
}

func login(c *gin.Context) {
	_, err := loggedAs(c)
	if err == nil {
		c.Redirect(http.StatusFound, "/")
		return
	}
	captchaNew(c)
	if err := renderLogin(c, ""); err != nil {
		internalError(c, err.Error())
		return
	}
}

func loginAs(c *gin.Context) {
	_, err := loggedAs(c)
	if err == nil {
		c.Redirect(http.StatusFound, "/")
		return
	}
	name := c.PostForm("username")
	password := c.PostForm("password")
	err = verifyCaptcha(c)
	var token string
	if err == nil {
		token, err = db.Login(name, password)
		if err != nil { err = errors.New("invalid credentials") }
	}
	if err != nil {
		captchaNew(c)
		if err := renderLogin(c, err.Error()); err != nil {
			internalError(c, err.Error())
			return
		}
		return
	}
	c.SetCookie("session_token", token, 0, "/", config.Cfg.Web.Domain,
			false, true)
	c.Redirect(http.StatusFound, "/")
}

func loggedAs(c *gin.Context) (db.Account, error) {
	token, _ := c.Cookie("session_token")
	if token == "" {
		return db.Account{}, errors.New("not logged in")
	}
	return db.GetAccountFromToken(token)
}

func disconnect(c *gin.Context) {
	token, err := c.Cookie("session_token")
	if err == nil || token != "" {
		db.Disconnect(token)
		c.SetCookie("session_token", "", 0, "/",
			config.Cfg.Web.Domain, false, true)
	}
	c.Redirect(http.StatusFound, "/")
}

func remove(c *gin.Context) {
	var err error
	for {
		var id int
		var post db.Post
		board := c.Param("board")
		id, err = strconv.Atoi(c.Param("id"))
		if err != nil { break }

		post, err = db.GetPostFromBoard(board, id)
		if err != nil { break }
		err = db.Remove(board, id)
		if err != nil { break }
		/*
		TODO: verify if media is used by another post
		if post.Media != "" {
			err = os.Remove(
				config.Cfg.Media.Directory + "/" + post.Media)
			if err != nil { break }
		}
		if post.Thumbnail() != "" {
			err = os.Remove(config.Cfg.Media.Thumbnail + "/" +
						post.Thumbnail())
			if err != nil { break }
		}
		*/

		dst := "/" + board
		if id != post.Thread.Number {
			dst += "/" + strconv.Itoa(post.Thread.Number)
		}
		c.Redirect(http.StatusFound, dst)
		return
	}
	badRequest(c, err.Error())
}

func hide(c *gin.Context) {
	var err error
	for {
		var id int
		var post db.Post
		id, err = strconv.Atoi(c.Param("id"))
		if err != nil { break }
		board := c.Param("board")
		post, err = db.GetPostFromBoard(board, id)
		if err != nil { break }
		err = db.Hide(board, id, post.Disabled)
		if err != nil { break }
		c.Redirect(http.StatusFound, "/" + board + "/" +
				strconv.Itoa(post.Thread.Number))
		return
	}
	badRequest(c, err.Error())
}

func Init() error {

	if err := os.MkdirAll(config.Cfg.Media.Directory, 0700); err != nil {
		return err
	}
	if err := os.MkdirAll(config.Cfg.Media.Thumbnail, 0700); err != nil {
		return err
	}

	r := gin.Default()
	if err := initTemplate(); err != nil { return err }

	r.GET("/", index)
	r.GET("/favicon.ico", func(c *gin.Context) {
		c.Data(http.StatusNotFound, "text/plain", []byte("Not Found"))
	})
	r.GET("/static/favicon.png", func(c *gin.Context) {
		c.Data(http.StatusOK, "image/png", favicon)
	})
	r.GET("/static/style.css", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/css", stylesheet)
	})
	if config.Cfg.Captcha.Enabled {
		r.GET("/captcha", captchaImage)
	}
	r.GET("/:board", boardIndex)
	r.GET("/:board/catalog", catalog)
	r.POST("/:board", newThread)
	r.GET("/:board/:thread", thread)
	r.POST("/:board/:thread", newPost)
	r.GET("/disconnect", disconnect)
	r.GET("/login", login)
	r.POST("/login", loginAs)
	r.GET("/:board/remove/:id", remove)
	r.GET("/:board/hide/:id", hide)
	r.GET("/:board/ban/:ip", login)
	r.Static("/media", mediaDir)
	r.Static("/thumbnail", thumbnailDir)

	return r.Run(config.Cfg.Web.Listener)
}
