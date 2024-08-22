package web

import (
	"strconv"
	"errors"

	"github.com/labstack/echo/v4"

	"IB1/db"
)

func badRequest(c echo.Context, info string) error {
	return render("error.html", info, c)
}

func renderFile(file string) echo.HandlerFunc {
	return func(c echo.Context) error {
		if err := render(file, nil, c); err != nil {
			return badRequest(c, err.Error())
		}
		return nil
	}
}

func boardIndex(c echo.Context) error {
	page, err := strconv.Atoi(c.QueryParam("page"))
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
	return render("board.html", data, c)
}

func catalog(c echo.Context) error {
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
	return render("catalog.html", board, c)
}

func thread(c echo.Context) error {

	var thread db.Thread
	var board db.Board

	threadID := c.Param("thread")
	boardName := c.Param("board")

	id, err := strconv.Atoi(threadID)
	if err != nil { return errors.New("file not found") }
	board, err = db.GetBoard(boardName)
	if err != nil { return err }
	thread, err = db.GetThread(board, id)
	if err != nil { return err }
	if thread.Posts[0].Disabled {
		if _, err := loggedAs(c); err != nil { return err }
	}
	return render("thread.html", thread, c)
}
