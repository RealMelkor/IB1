package web

import (
	"github.com/labstack/echo/v4"
	"github.com/gabriel-vasile/mimetype"
	"errors"
	"strings"

	"IB1/config"
	"IB1/db"
)

func updateFavicon(c echo.Context) error {
	file, err := c.FormFile("theme")
        if err != nil { return err }
	f, err := file.Open()
        if err != nil { return err }
	defer f.Close()
	data := make([]byte, file.Size)
	_, err = f.Read(data)
        if err != nil { return err }

	mime := mimetype.Detect(data)
	if strings.Index(mime.String(), "image/") != 0 {
		return errors.New("invalid mime type")
	}
	config.Cfg.Home.FaviconMime = mime.String()
	config.Cfg.Home.Favicon = data
	db.UpdateConfig()
	return nil
}

func clearFavicon(c echo.Context) error {
	config.Cfg.Home.FaviconMime = ""
	config.Cfg.Home.Favicon = nil
	db.UpdateConfig()
	return nil
}
