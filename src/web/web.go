package web

import (
	"github.com/labstack/echo/v4"
	"net/http"
	"hash/fnv"
	"os"
	"strings"
	"time"
	"strconv"
	"log"
	
	"IB1/db"
	"IB1/config"
)

func clientIP(c echo.Context) string {
	ip := c.Request().Header.Get("X-Real-IP")
	if ip == "" { return c.Request().RemoteAddr }
	return ip
}

func err(f echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if err := f(c); err != nil {
			return badRequest(c, err.Error())
		}
		return nil
	}
}

func logger(next echo.HandlerFunc) echo.HandlerFunc {
	return func (c echo.Context) error {
		t1 := time.Now()
		id, err := getID(c)
		if id != "" && err == nil {
			h := fnv.New32()
			h.Write([]byte(id))
			id = strconv.Itoa(int(h.Sum32()))
		}
		err = next(c)
		t2 := time.Now()
		r := c.Request()
		ip := clientIP(c)
		log.Println("[" + id + "][" + ip + "][" + r.Method + "]",
		r.URL.String(), t2.Sub(t1))
		return err
	}
}

func isBanned(c echo.Context) error {
	_, err := loggedAs(c)
	if err == nil { return err }
	return db.IsBanned(clientIP(c))
}

func hasRank(f echo.HandlerFunc, rank int) echo.HandlerFunc {
	return func(c echo.Context) error {
		if err := needRank(c, rank); err != nil { return err }
		return f(c)
	}
}

func catchCustom(f echo.HandlerFunc, param string,
			redirect string) echo.HandlerFunc {
	return func(c echo.Context) error {
		if err := f(c); err != nil {
			set(c)(param, err.Error())
			c.Redirect(http.StatusFound, redirect)
		}
		return nil
	}
}

func catch(f echo.HandlerFunc, param string) echo.HandlerFunc {
	return func(c echo.Context) error {
		return catchCustom(f, param, c.Request().RequestURI)(c)
	}
}

func unauth(f echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		_, err := loggedAs(c)
		if err == nil {
			c.Redirect(http.StatusFound, "/")
			return nil
		}
		return f(c)
	}
}

func notFound(c echo.Context) error {
	return c.Blob(http.StatusNotFound, "text/plain", []byte("Not Found"))
}

func Init() error {

	if !config.Cfg.Media.InDatabase {
		os.MkdirAll(config.Cfg.Media.Path + "/thumbnail", 0700)
	}
	os.MkdirAll(config.Cfg.Media.Tmp, 0700)

	r := echo.New()
	if err := initTemplate(); err != nil { return err }

	r.Use(logger)
	r.Use(err)

	r.GET("/", renderFile("index.html"))
	r.GET("/favicon.ico", notFound)
	r.GET("/static/favicon", func(c echo.Context) error {
		if config.Cfg.Home.FaviconMime == "" {
			return c.Blob(http.StatusOK, "image/png", favicon)
		}
		return c.Blob(http.StatusOK, config.Cfg.Home.FaviconMime,
				config.Cfg.Home.Favicon)
	})
	r.GET("/static/common.css", func(c echo.Context) error {
		return c.Blob(http.StatusOK, "text/css", stylesheet)
	})
	r.GET("/static/:file", func(c echo.Context) error {
		content, ok := themesContent[c.Param("file")]
		if !ok { return notFound(c) }
		c.Response().Writer.Header().Add("Content-Type", "text/css")
		c.Response().Writer.Write(content)
		return nil
	})
	if config.Cfg.Captcha.Enabled {
		r.GET("/captcha", captchaImage)
	}
	r.GET("/:board", boardIndex)
	r.GET("/:board/catalog", catalog)
	r.POST("/:board", catch(newThread, "new-thread-error"))
	r.GET("/:board/:thread", thread)
	r.POST("/:board/:thread", catch(newPost, "new-post-error"))
	r.GET("/disconnect", disconnect)
	r.GET("/login", unauth(renderFile("login.html")))
	r.POST("/login", unauth(catch(loginAs, "login-error")))
	r.GET("/:board/remove/:id", hasRank(remove, db.RANK_ADMIN))
	r.GET("/:board/hide/:id", hasRank(hide, db.RANK_MODERATOR))
	r.GET("/:board/ban/:ip", hasRank(ban, db.RANK_MODERATOR))
	r.GET("/dashboard", hasRank(renderDashboard, db.RANK_ADMIN))
	r.POST("/config/client/theme", func(c echo.Context) error {
		return redirect(setTheme, c.QueryParam("origin"))(c)
	})
	r.POST("/config/update", handleConfig(updateConfig, "config-error"))
	r.POST("/config/board/create",
		handleConfig(createBoard, "board-error"))
	r.POST("/config/board/update/:board",
		handleConfig(updateBoard, "board-error"))
	r.POST("/config/board/delete/:board",
		handleConfig(deleteBoard, "board-error"))
	r.POST("/config/theme/create",
		handleConfig(createTheme, "theme-error"))
	r.POST("/config/theme/delete/:id",
		handleConfig(deleteTheme, "theme-error"))
	r.POST("/config/theme/update/:id",
		handleConfig(updateTheme, "theme-error"))
	r.POST("/config/favicon/update",
		handleConfig(updateFavicon, "favicon-error"))
	r.POST("/config/favicon/clear",
		handleConfig(clearFavicon, "favicon-error"))
	r.POST("/config/ban/create", handleConfig(addBan, "ban-error"))
	r.POST("/config/ban/cancel/:id", handleConfig(deleteBan, "ban-error"))

	if config.Cfg.Media.InDatabase {
		r.GET("/media/:hash", func(c echo.Context) error {
			parts := strings.Split(c.Param("hash"), ".")
			data, mime, err := db.GetMedia(parts[0])
			if err != nil { return err }
			return c.Blob(http.StatusOK, mime, data)
		})
		r.GET("/media/thumbnail/:hash", func(c echo.Context) error {
			parts := strings.Split(c.Param("hash"), ".")
			data, err := db.GetThumbnail(parts[0])
			if err != nil { return err }
			return c.Blob(http.StatusOK, "image/png", data)
		})
	} else {
		r.Static("/media", config.Cfg.Media.Path)
	}

	return r.Start(config.Cfg.Web.Listener)
}
