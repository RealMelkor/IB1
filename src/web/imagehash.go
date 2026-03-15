package web

import (
	"github.com/labstack/echo/v4"
	"strconv"

	"IB1/db"
)

func addBannedHash(c echo.Context) error {
	hash, err := strconv.Atoi(c.FormValue("hash"))
	if err != nil {
		return err
	}
	return db.AddBannedImage(int64(hash))
}

func removeBannedHash(c echo.Context) error {
	hash, err := strconv.Atoi(c.FormValue("hash"))
	if err != nil {
		return err
	}
	return db.RemoveBannedImage(int64(hash))
}
