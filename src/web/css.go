package web

import (
	"strconv"
	"github.com/labstack/echo/v4"

	"IB1/db"
)

func getThreads(c echo.Context) ([]db.Thread, error) {
	if c.Param("board") == "" {
		return nil, nil
	}
	board, err := db.GetBoard(c.Param("board"))
	if err != nil {
		return nil, err
	}
	if c.Param("thread") == "" {
		threads, err := db.GetVisibleThreads(board)
		if err != nil {
			return nil, err
		}
		for i := range threads {
			threads[i].Board = board
		}
		return threads, err
	}
	id, err := strconv.Atoi(c.Param("thread"))
	if err != nil {
		return nil, err
	}
	thread, err := db.GetThread(board, id)
	if err != nil {
		return nil, err
	}
	thread.Board = board
	return []db.Thread{thread}, nil
}

func threadCSS(c echo.Context) error {
	board, err := db.GetBoard(c.Param("board"))
	if err != nil {
		return err
	}
	id, err := strconv.Atoi(c.Param("thread"))
	if err != nil {
		return err
	}
	thread, err := db.GetThread(board, id)
	if err != nil {
		return err
	}
	return renderContent("thread.css", thread, c, "text/css", true)
}
