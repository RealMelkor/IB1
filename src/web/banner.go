package web

import (
	"github.com/labstack/echo/v4"
	"strconv"

	"IB1/db"
)

func banner(c echo.Context) error {
	v, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return err
	}
	banner, err := db.GetBanner(uint(v))
	if err != nil {
		return err
	}
	serveMedia(c, banner.Data, "banner")
	return nil
}
