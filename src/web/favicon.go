package web

import (
	"errors"
	"github.com/gabriel-vasile/mimetype"
	"github.com/labstack/echo/v4"
	"strings"

	"IB1/config"
	"IB1/db"
)

func handleImage(c echo.Context, param string) ([]byte, string, error) {
	file, err := c.FormFile(param)
	if err != nil {
		return nil, "", err
	}
	f, err := file.Open()
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	data := make([]byte, file.Size)
	_, err = f.Read(data)
	if err != nil {
		return nil, "", err
	}

	mime := mimetype.Detect(data)
	if strings.Index(mime.String(), "image/") != 0 {
		return nil, "", errors.New("invalid mime type")
	}
	return data, mime.String(), nil
}

func updateFavicon(c echo.Context) error {
	data, mime, err := handleImage(c, "favicon")
	if err != nil {
		return err
	}
	config.Cfg.Home.FaviconMime = mime
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
