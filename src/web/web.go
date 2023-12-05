package web

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"log"
	
	"IB1/db"
)

func html(c *gin.Context, code int, data string) {
	c.Data(code, "text/html; charset=utf-8",
		[]byte(header + data + footer))
}

func htmlOK(c *gin.Context, data string) {
	html(c, http.StatusOK, data)
}

/*func htmlInternalError(c *gin.Context, data string) {
	html(c, http.StatusInternalServerError, data)
}

func htmlBadRequest(c *gin.Context, data string) {
	html(c, http.StatusBadRequest, data)
}*/

func internalError(c *gin.Context, data string) {
	c.Data(http.StatusBadRequest, "text/plain", []byte(data))
}

func badRequest(c *gin.Context, data string) {
	log.Println(data)
	c.Data(http.StatusBadRequest, "text/plain", []byte("bad request"))
}

func index(c *gin.Context) {
	res, err := minifyIndex()
	if err != nil {
		internalError(c, err.Error())
		return
	}
	c.Data(http.StatusOK, "text/html", res)
}

func boardIndex(c *gin.Context) {
	boardName := c.Param("board")
	board, err := db.GetBoard(boardName)
	if err != nil { 
		internalError(c, err.Error())
		return
	}
	res, err := renderBoard(board)
	if err != nil { 
		internalError(c, err.Error())
		return
	}
	htmlOK(c, res)
}

func catalog(c *gin.Context) {
	boardName := c.Param("board")
	board, err := db.GetBoard(boardName)
	if err != nil { 
		internalError(c, err.Error())
		return
	}
	for i, v := range board.Threads {
		post, err := db.GetPost(v, v.Number)
		if err != nil {
			internalError(c, err.Error())
		}
		board.Threads[i].Posts = append(board.Threads[i].Posts, post)
	}
	catalog, err := renderCatalog(board)
	if err != nil { 
		internalError(c, err.Error())
		return
	}
	htmlOK(c, catalog)
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
		badRequest(c, "invalid form")
		return 
	}

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

	number, err := db.CreateThread(board, title, name,
				media, parseContent(content))
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

	media := ""
	file, err := c.FormFile("media")
	if err == nil { 
		if media, err = uploadFile(c, file); err != nil { 
			badRequest(c, err.Error())
			return
		}
	}

	_, err = db.CreatePost(thread, parseContent(content), name, media, nil)
	if err != nil {
		internalError(c, err.Error())
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
	thread.Posts[0].Title = thread.Title
	data, err = renderThread(thread)
	if err != nil { 
		internalError(c, err.Error())
		return
	}
	htmlOK(c, data)
}

func Init() error {

	r := gin.Default()
	if err := initTemplate(); err != nil { return err }

	r.GET("/", index)
	r.GET("/favicon.ico", func(c *gin.Context) {
		c.Data(http.StatusNotFound, "text/plain", []byte("Not Found"))
	})
	r.GET("/static/favicon.png", func(c *gin.Context) {
		c.Data(http.StatusOK, "image/png", []byte(favicon))
	})
	r.GET("/static/style.css", func(c *gin.Context) {
		b, err := minifyStylesheet()
		if err != nil {
			c.Data(http.StatusInternalServerError, "text/plain",
				[]byte(err.Error()))
			return
		}
		c.Data(http.StatusOK, "text/css", b)
	})
	r.GET("/:board", boardIndex)
	r.GET("/:board/catalog", catalog)
	r.POST("/:board", newThread)
	r.GET("/:board/:thread", thread)
	r.POST("/:board/:thread", newPost)
	r.Static("/media", mediaDir)
	r.Static("/thumbnail", thumbnailDir)

	return r.Run(":8080")
}
