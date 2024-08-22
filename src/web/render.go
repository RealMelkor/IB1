package web

import (
	"strconv"
	"net/http"

	"github.com/gin-gonic/gin"

	"IB1/db"
)

func renderCustom(c *gin.Context, file string, data any) error {
	var v = map[string]any{
		"Header": header(c),
		"Data": data,
	}
	return render(file, v, c)
}

func badRequest(c *gin.Context, info string) {
	renderCustom(c, "error.html", info)
}

func renderFile(file string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := render(file, header(c), c); err != nil {
			badRequest(c, err.Error())
		}
	}
}

func internalError(c *gin.Context, data string) {
	c.Data(http.StatusBadRequest, "text/plain", []byte(data))
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
	pages := []int{}
	count := (threads + 3) / 4
	for i := 0; i < count; i++ { pages = append(pages, i + 1) }
	data := struct {
		Board	db.Board
		Pages	[]int
	}{
		Board: board,
		Pages: pages,
	}
	return renderCustom(c, "board.html", data)
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
	return renderCustom(c, "catalog.html", board)
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
	return renderCustom(c, "thread.html", thread)
}

