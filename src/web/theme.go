package web

import (
	"strings"
	"errors"
	"IB1/config"
	"IB1/db"

	"github.com/gin-gonic/gin"
)

var themes []string
var themesTable map[string]bool
var themesContent map[string][]byte
func getThemes() []string {
	if themes != nil { return themes }
	themesTable = map[string]bool{}
	files, err := static.ReadDir("static")
	if err != nil { return []string{} }
	themes = []string{}
	themesContent = map[string][]byte{}
	for _, v := range files {
		if !v.Type().IsRegular() || v.Name() == "common.css" {
			continue
		}
		parts := strings.Split(v.Name(), ".")
		theme := parts[len(parts) - 2]
		if len(parts) < 2  || parts[len(parts) - 1] != "css"{
			continue
		}
		themesTable[theme] = true
		data, err := static.ReadFile("static/" + v.Name())
		if err != nil { continue }
		data, err = minifyCSS(data)
		if err != nil { continue }
		themesContent[v.Name()] = data
		themes = append(themes, theme)
	}
	dbThemes, err := db.GetThemes()
	if err == nil {
		for _, v := range dbThemes {
			if v.Disabled { continue }
			themes = append(themes, v.Name)
			themesTable[v.Name] = true
			themesContent[v.Name + ".css"] = []byte(v.Content)
		}
	}
	return themes
}

func getThemesTable() map[string]bool {
	getThemes()
	return themesTable
}

func reloadThemes() {
	themes = nil
	getThemes()
}

func getTheme(c *gin.Context) string {
	theme, err := c.Cookie("theme")
	if err != nil { return config.Cfg.Home.Theme }
	_, ok := getThemesTable()[theme]
	if !ok { return config.Cfg.Home.Theme }
	return theme
}

func setTheme(c *gin.Context) error {
	theme, ok := c.GetPostForm("theme")
	if !ok { return errors.New("invalid form") }
	_, ok = themesTable[theme]
	if !ok { return errors.New("invalid theme") }
	c.SetCookie("theme", theme, 315360000, "", config.Cfg.Web.Domain,
			true, false)
	return nil
}
