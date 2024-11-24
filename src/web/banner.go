package web

import (
	"strconv"
	"github.com/labstack/echo/v4"

	"IB1/db"
)

func addBanner(c echo.Context) error {
	file, err := c.FormFile("banner")
        if err != nil { return err }
	data := make([]byte, file.Size)
	f, err := file.Open()
	if err != nil { return err }
	defer f.Close()
	_, err = f.Read(data)
	if err != nil { return err }
	return db.AddBanner(data)
}

func deleteBanner(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil { return err }
	return db.RemoveBanner(id)
}

func banner(c echo.Context) error {
	v, err := strconv.Atoi(c.Param("id"))
	if err != nil { return err }
	banner, err := db.GetBanner(uint(v))
	if err != nil { return err }
	serveMedia(c, banner.Data, "banner")
	return nil
}
