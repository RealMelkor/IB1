package web

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"log"
	"strconv"
	
	"IB1/db"
)

func html(c *gin.Context, code int, data string) {
	c.Data(code, "text/html; charset=utf-8",
		[]byte(header + data + footer))
}

func htmlOK(c *gin.Context, data string) {
	html(c, http.StatusOK, data)
}

func htmlInternalError(c *gin.Context, data string) {
	html(c, http.StatusInternalServerError, data)
}

func htmlBadRequest(c *gin.Context, data string) {
	html(c, http.StatusBadRequest, data)
}

func index(c *gin.Context) {
	htmlOK(c, "")
}

func catalog(c *gin.Context) {
	boardName := c.Param("board")
	board, err := db.GetBoard(boardName)
	if err != nil { 
		htmlInternalError(c, err.Error())
		return
	}
	catalog, err := renderCatalog(board)
	if err != nil { 
		htmlInternalError(c, err.Error())
		return
	}
	htmlOK(c, catalog)
}

func newThread(c *gin.Context) {

	boardName := c.Param("board")
	board, err := db.GetBoard(boardName)
	if err != nil { 
		htmlInternalError(c, err.Error())
		return 
	}

	title, hasTitle := c.GetPostForm("title")
	content, hasContent := c.GetPostForm("content")
	if !hasTitle || !hasContent { 
		htmlBadRequest(c, "invalid form")
		return 
	}

	number, err := db.CreateThread(board, title, content)
	if err != nil { 
		htmlInternalError(c, err.Error())
		return
	}

	c.Redirect(http.StatusFound, c.Request.URL.Path + "/" +
			strconv.Itoa(number))
}

func newPost(c *gin.Context) {

	boardName := c.Param("board")
	board, err := db.GetBoard(boardName)
	if err != nil { 
		htmlBadRequest(c, err.Error())
		return
	}

	threadNumberStr := c.Param("thread")
	threadNumber, err := strconv.Atoi(threadNumberStr)
	if err != nil { 
		htmlBadRequest(c, err.Error())
		return
	}
	thread, err := db.GetThread(board, threadNumber)
	if err != nil {
		htmlBadRequest(c, err.Error())
		return
	}

	content, exist := c.GetPostForm("content")
	if !exist {
		htmlBadRequest(c, "invalid form")
		return
	}

	media := ""
	file, err := c.FormFile("media")
	if err == nil { 
		if media, err = uploadFile(c, file); err != nil { 
			htmlBadRequest(c, err.Error())
			return
		}
	}
	
	if _, err = db.CreatePost(thread, content, media, nil); err != nil {
		htmlInternalError(c, err.Error())
		return
	}

	c.Redirect(http.StatusFound, c.Request.URL.Path)
}

func thread(c *gin.Context) {

	var data string
	var thread db.Thread
	var board db.Board

	threadID := c.Param("thread")
	boardName := c.Param("board")

	id, err := strconv.Atoi(threadID)
	if err != nil {
		htmlBadRequest(c, err.Error())
		return
	}
	board, err = db.GetBoard(boardName)
	if err != nil {
		htmlInternalError(c, err.Error())
		return
	}
	thread, err = db.GetThread(board, id)
	if err != nil {
		htmlInternalError(c, err.Error())
		return
	}
	data, err = renderThread(thread)
	if err != nil { 
		htmlInternalError(c, err.Error())
		return
	}
	htmlOK(c, data)
}

func Init() error {

	r := gin.Default()
	if err := initTemplate(); err != nil { return err }

	r.GET("/", index)
	r.GET("/static/favicon.png", func(c *gin.Context) {
		c.Data(200, "image/png", []byte(favicon))
	})
	r.GET("/static/style.css", func(c *gin.Context) {
		c.Data(200, "text/css", []byte(stylesheet))
	})
	r.GET("/:board", catalog)
	r.POST("/:board", newThread)
	r.GET("/:board/:thread", thread)
	r.POST("/:board/:thread", newPost)
	r.Static("/media", "./media")

        log.Println("web server started")
	return r.Run(":8080")
}
