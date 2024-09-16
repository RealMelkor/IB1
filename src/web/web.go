package web

import (
	"github.com/labstack/echo/v4"
	"github.com/gabriel-vasile/mimetype"
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
	if ip == "" {
		s := c.Request().RemoteAddr
		s = strings.Replace(s, "[", "", 1)
		s = strings.Replace(s, "]", "", 1)
		return s[:strings.LastIndex(s, ":")]
	}
	return ip
}

func fatalError(c echo.Context, err error) {
	c.Response().Write([]byte("FATAL ERROR: " + err.Error()))
}

func imageError(f echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if err := f(c); err == nil { return nil }
		c.Response().Writer.Header().Add("Content-Type", "image/png")
		c.Response().WriteHeader(http.StatusOK)
		c.Response().Write(mediaError)
		return nil
	}
}

func mediaCheck(f echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		for config.Cfg.Media.ApprovalQueue {
			user, err := loggedAs(c)
			if err == nil && user.HasRank("moderator") { break }
			err = db.IsApproved(
				strings.Split(c.Param("hash"), ".")[0])
			if err == nil { break }
			c.Response().Writer.Header().Add(
                		"Content-Type", "image/png")
			c.Response().WriteHeader(http.StatusOK)
			c.Response().Write(pendingMedia)
			return nil
		}
		return f(c)
	}
}

func approveMedia(c echo.Context) error {
	return db.Approve(strings.Split(c.FormValue("media"), ".")[0])
}

func denyMedia(c echo.Context) error {
	return db.RemoveMedia(strings.Split(c.FormValue("media"), ".")[0])
}

func err(f echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if err := f(c); err != nil {
			if (len(c.Response().Header()) == 0) {
				return badRequest(c, err)
			}
			fatalError(c, err)
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
		name := ""
		user, err := loggedAs(c)
		if err == nil { name = "[" + user.Name + "]" }
		err = next(c)
		t2 := time.Now()
		r := c.Request()
		ip := clientIP(c)
		log.Println(
			"[" + id + "][" + ip + "][" + r.Method + "]" + name,
			r.URL.String(), t2.Sub(t1))
		return err
	}
}

func isBanned(c echo.Context) error {
	_, err := loggedAs(c)
	if err == nil { return nil }
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
	r.POST("/:board", catch(readOnly(newThread), "new-thread-error"))
	r.GET("/:board/:thread", thread)
	r.POST("/:board/:thread", catch(readOnly(newPost), "new-post-error"))
	r.GET("/disconnect", disconnect)
	r.GET("/login", unauth(renderFile("login.html")))
	r.POST("/login", unauth(catch(loginAs, "login-error")))
	r.GET("/register", unauth(renderFile("register.html")))
	r.POST("/register", unauth(catch(readOnly(register), "register-error")))
	r.GET("/:board/cancel/:id", cancel)
	r.GET("/:board/remove/:id", hasRank(remove, db.RANK_ADMIN))
	r.GET("/:board/hide/:id", hasRank(hide, db.RANK_MODERATOR))
	r.GET("/:board/ban/:ip", hasRank(ban, db.RANK_MODERATOR))
	if config.Cfg.Media.ApprovalQueue {
		r.GET("/approval", hasRank(
			renderFile("approval.html"), db.RANK_MODERATOR))
		r.POST("/approval/accept", hasRank(redirect(
			approveMedia, "/approval"), db.RANK_MODERATOR))
		r.POST("/approval/deny", hasRank(redirect(
			denyMedia, "/approval"), db.RANK_MODERATOR))
	}
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
	r.POST("/config/account/create",
		handleConfig(addAccount, "account-error"))
	r.POST("/config/account/update/:id",
		handleConfig(updateAccount, "account-error"))
	r.POST("/config/account/delete/:id",
		handleConfig(deleteAccount, "account-error"))
	r.POST("/config/restart",
		handleConfig(restart, "config-error"))

	if config.Cfg.Media.InDatabase {
		r.GET("/media/:hash", imageError(mediaCheck(
			func(c echo.Context) error {
				parts := strings.Split(c.Param("hash"), ".")
				data, mime, err := db.GetMedia(parts[0])
				if err != nil { return err }
				return c.Blob(http.StatusOK, mime, data)
			})))
		r.GET("/media/thumbnail/:hash", imageError(mediaCheck(
			func(c echo.Context) error {
				parts := strings.Split(c.Param("hash"), ".")
				data, err := db.GetThumbnail(parts[0])
				if err != nil { return err }
				return c.Blob(http.StatusOK, "image/png", data)
			})))
	} else if config.Cfg.Media.ApprovalQueue {
		f := func(c echo.Context) error {
			path := c.Request().RequestURI
			path = strings.TrimPrefix(path, "/media")
			path = config.Cfg.Media.Path + path
			m, err := mimetype.DetectFile(path)
			if err != nil { return err }
			c.Response().Writer.Header().Add(
					"Content-Type", m.String())
			c.Response().WriteHeader(http.StatusOK)
			f, err := os.Open(path)
			if err != nil { return err }
			for {
				var buf [4096]byte
				n, err := f.Read(buf[:])
				if err != nil {
					if err.Error() == "EOF" { break }
					return err
				}
				c.Response().Write(buf[:n])
				if n != len(buf) { break }
			}
			return nil
		}
		r.GET("/media/:hash", imageError(mediaCheck(f)))
		r.GET("/media/thumbnail/:hash", imageError(mediaCheck(f)))

	} else {
		r.Static("/media", config.Cfg.Media.Path)
	}

	if s := os.Getenv("IB1_LISTENER"); s != "" {
		return r.Start(s)
	}
	return r.Start(config.Cfg.Web.Listener)
}
