package web

import (
	"github.com/gin-gonic/gin"
	"html/template"
	"net/http"
	"strconv"
	"os"
	"strings"
	"errors"
	
	"IB1/db"
	"IB1/config"
)

type fErr func(c *gin.Context) error

func render(_template string, data any, c *gin.Context) error {
	c.Writer.WriteHeader(http.StatusOK)
	c.Writer.Header().Add("Content-Type", "text/html; charset=utf-8")
	w := minifyHTML(c.Writer)
	funcs := template.FuncMap{
		"get": get(c),
		"set": set(c),
	}
	err := templates.Funcs(funcs).Lookup(_template).Execute(w, data)
	if err != nil { return err }
	w.Close()
	return nil
}

func internalError(c *gin.Context, data string) {
	c.Data(http.StatusBadRequest, "text/plain", []byte(data))
}

func badRequest(c *gin.Context, info string) {
	data := struct {
		Error	string
		Header	any
	}{
		Error: info,
		Header: header(c),
	}
	render("error.gohtml", data, c)
}

func isBanned(c *gin.Context) error {
	_, err := loggedAs(c)
	if err == nil { return err }
	return db.IsBanned(c.RemoteIP())
}

func hasRank(f fErr, rank int) fErr {
	return func(c *gin.Context) error {
		if err := needRank(c, rank); err != nil { return err }
		return f(c)
	}
}

func err(f func(c *gin.Context) error) func(c *gin.Context) {
	return func(c *gin.Context) {
		if err := f(c); err != nil {
			badRequest(c, err.Error())
			return
		}
	}
}

func boardIndex(c *gin.Context) error {
	page, err := strconv.Atoi(c.Query("page"))
	if err != nil { page = 0 } else { page -= 1 }
	boardName := c.Param("board")
	board, err := db.GetBoard(boardName)
	if err != nil { return err }
	account, err := loggedAs(c)
	if err == nil && account.Rank < db.RANK_MODERATOR {
		board.Threads, err = db.GetVisibleThreads(board)
		if err != nil { return err }
	}
	threads := len(board.Threads)
	if threads > 4 {
		if page < 0 || page * 4 >= threads { page = 0 }
		i := 4
		if threads % 4 != 0 && page * 4 + i >= threads {
			i = threads % 4
		}
		board.Threads = board.Threads[page * 4 : page * 4 + i]
	}
	for i := range board.Threads {
		if err := db.RefreshThread(&board.Threads[i]); err != nil {
			return err
		}
		if length := len(board.Threads[i].Posts); length > 5 {
			posts := []db.Post{board.Threads[i].Posts[0]}
			board.Threads[i].Posts = append(posts,
				board.Threads[i].Posts[length - 4:]...)
		}
	}
	return renderBoard(board, threads, c)
}

func catalog(c *gin.Context) error {
	boardName := c.Param("board")
	board, err := db.GetBoard(boardName)
	if err != nil { return err }
	for i, v := range board.Threads {
		err := db.RefreshThread(&v)
		if err != nil { return err }
		v.Replies = len(v.Posts) - 1
		v.Images = -1
		for _, post := range v.Posts {
			if post.Media != "" {
				v.Images++
			}
		}
		board.Threads[i] = v
	}
	return renderCatalog(board, c)
}

func checkCaptcha(c *gin.Context) error {
	if config.Cfg.Captcha.Enabled {
		_, err := loggedAs(c)
		if err == nil { return nil } // captcha not needed if logged
		captcha, hasCaptcha := c.GetPostForm("captcha")
		if !hasCaptcha {
			return errors.New("invalid form")
		}
		if !captchaVerify(c, captcha) {
			return errors.New("wrong captcha")
		}
	}
	return nil
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

func thread(c *gin.Context) error {

	var thread db.Thread
	var board db.Board

	threadID := c.Param("thread")
	boardName := c.Param("board")

	id, err := strconv.Atoi(threadID)
	if err != nil { return err }
	board, err = db.GetBoard(boardName)
	if err != nil { return err }
	thread, err = db.GetThread(board, id)
	if err != nil { return err }
	if thread.Posts[0].Disabled {
		if _, err := loggedAs(c); err != nil { return err }
	}
	return renderThread(thread, c)
}

func login(c *gin.Context) error {
	_, err := loggedAs(c)
	if err == nil {
		c.Redirect(http.StatusFound, "/")
		return nil
	}
	return renderLogin(c)
}

func loginAs(c *gin.Context) error {
	_, err := loggedAs(c)
	if err == nil {
		c.Redirect(http.StatusFound, "/")
		return nil
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
		set(c)("login-error", err.Error())
		c.Redirect(http.StatusFound, "/login")
		return nil
	}
	c.SetCookie("session_token", token, 0, "/", config.Cfg.Web.Domain,
			false, true)
	c.Redirect(http.StatusFound, "/")
	return nil
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

func remove(c *gin.Context) error {
	board := c.Param("board")
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil { return err }

	post, err := db.GetPostFromBoard(board, id)
	if err != nil { return err }
	err = db.Remove(board, id)
	if err != nil { return err }
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

func Init() error {

	if !config.Cfg.Media.InDatabase {
		os.MkdirAll(config.Cfg.Media.Path + "/thumbnail", 0700)
	}
	os.MkdirAll(config.Cfg.Media.Tmp, 0700)

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	if err := initTemplate(); err != nil { return err }

	r.GET("/", err(renderIndex))
	r.GET("/favicon.ico", func(c *gin.Context) {
		c.Data(http.StatusNotFound, "text/plain", []byte("Not Found"))
	})
	r.GET("/static/favicon", func(c *gin.Context) {
		if config.Cfg.Home.FaviconMime == "" {
			c.Data(http.StatusOK, "image/png", favicon)
			return
		}
		c.Data(http.StatusOK, config.Cfg.Home.FaviconMime,
			config.Cfg.Home.Favicon)
	})
	r.GET("/static/common.css", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/css", stylesheet)
	})
	r.GET("/static/:file", func(c *gin.Context) {
		content, ok := themesContent[c.Param("file")]
		if !ok {
			internalError(c, "file not found")
			return
		}
		c.Writer.Header().Add("Content-Type", "text/css")
		c.Writer.Write(content)
	})
	if config.Cfg.Captcha.Enabled {
		r.GET("/captcha", captchaImage)
	}
	r.GET("/:board", err(boardIndex))
	r.GET("/:board/catalog", err(catalog))
	r.POST("/:board", err(newThread))
	r.GET("/:board/:thread", err(thread))
	r.POST("/:board/:thread", err(newPost))
	r.GET("/disconnect", disconnect)
	r.GET("/login", err(login))
	r.POST("/login", err(loginAs))
	r.GET("/:board/remove/:id", err(hasRank(remove, db.RANK_ADMIN)))
	r.GET("/:board/hide/:id", err(hasRank(hide, db.RANK_MODERATOR)))
	r.GET("/:board/ban/:ip", err(hasRank(ban, db.RANK_MODERATOR)))
	r.GET("/dashboard", err(hasRank(renderDashboard, db.RANK_ADMIN)))
	r.POST("/config/client/theme", func(c *gin.Context) {
		redirect(setTheme, c.Query("origin"))(c)
	})
	r.POST("/config/update", handleConfig(updateConfig))
	r.POST("/config/board/create", handleConfig(createBoard))
	r.POST("/config/board/update/:board", handleConfig(updateBoard))
	r.POST("/config/board/delete/:board", handleConfig(deleteBoard))
	r.POST("/config/theme/create", handleConfig(createTheme))
	r.POST("/config/theme/delete/:id", handleConfig(deleteTheme))
	r.POST("/config/theme/update/:id", handleConfig(updateTheme))
	r.POST("/config/favicon/update", handleConfig(updateFavicon))
	r.POST("/config/favicon/clear", handleConfig(clearFavicon))
	r.POST("/config/ban/create", handleConfig(addBan))
	r.POST("/config/ban/cancel/:id", handleConfig(deleteBan))

	if config.Cfg.Media.InDatabase {
		r.GET("/media/:hash", func(c *gin.Context) {
			parts := strings.Split(c.Param("hash"), ".")
			data, mime, err := db.GetMedia(parts[0])
			if err != nil {
				c.Data(http.StatusBadRequest, "text/plain",
						[]byte(err.Error()))
				return
			}
			c.Data(http.StatusOK, mime, data)
		})
		r.GET("/media/thumbnail/:hash", func(c *gin.Context) {
			parts := strings.Split(c.Param("hash"), ".")
			data, err := db.GetThumbnail(parts[0])
			if err != nil {
				c.Data(http.StatusBadRequest, "text/plain",
						[]byte(err.Error()))
				return
			}
			c.Data(http.StatusOK, "image/png", data)
		})
	} else {
		r.Static("/media", config.Cfg.Media.Path)
	}

	return r.Run(config.Cfg.Web.Listener)
}
