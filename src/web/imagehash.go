package web

import (
	"io"
	"image"
	_ "image/jpeg"
	_ "image/png"
	_ "image/gif"
	"strconv"
	"github.com/corona10/goimagehash"
	"github.com/labstack/echo/v4"

	"IB1/db"
)

func isImageBanned(r io.Reader) error {
	img, _, err := image.Decode(r)
	if err != nil { return err }
	v, err := goimagehash.AverageHash(img)
	if err != nil { return err }
	return db.IsImageBanned(*v)
}

func banImage(hash string) error {
	r, err := mediaReader(hash)
	if err != nil { return err }
	img, _, err := image.Decode(r)
	if err != nil { return err }
	v, err := goimagehash.AverageHash(img)
	if err != nil { return err }
	return db.BanImage(*v)
}

func addBannedHash(c echo.Context) error {
	hash, err := strconv.Atoi(c.FormValue("hash"))
	if err != nil { return err }
	return db.AddBannedImage(int64(hash))
}

func removeBannedHash(c echo.Context) error {
	hash, err := strconv.Atoi(c.FormValue("hash"))
	if err != nil { return err }
	return db.RemoveBannedImage(int64(hash))
}
