package web

import (
	"github.com/labstack/echo/v4"
	"net/http"
	"hash/fnv"
	"bytes"
	"os"
	"strings"
	"time"
	"strconv"
	"log"
	
	"IB1/db"
	"IB1/config"
)

func serveMedia(c echo.Context, data []byte, name string) {
	r := bytes.NewReader(data)
	http.ServeContent(c.Response().Writer,
	c.Request(), name, time.Now(), r)
}

func clientIP(c echo.Context) string {
	ip := c.Request().Header.Get("X-Real-IP")
	if ip == "" {
		s := c.Request().RemoteAddr
		s = strings.Replace(s, "[", "", 1)
		s = strings.Replace(s, "]", "", 1)
		i := strings.LastIndex(s, ":")
		if i == -1 { return s }
		return s[:i]
	}
	return ip
}

func fatalError(c echo.Context, err error) {
	c.Response().Write([]byte("FATAL ERROR: " + err.Error()))
}

func imageError(f echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if err := f(c); err == nil { return nil }
		serveMedia(c, mediaError, "media")
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
			if err.Error() != db.NoYetApproved { return err }
			return pendingMediaImage(c)
		}
		return f(c)
	}
}

func approveMedia(c echo.Context) error {
	return db.Approve(strings.Split(c.FormValue("media"), ".")[0])
}

func approveAll(c echo.Context) error {
	return db.ApproveAll()
}

func denyMedia(c echo.Context) error {
	return db.RemoveMedia(strings.Split(c.FormValue("media"), ".")[0])
}

func banPendingMedia(c echo.Context) error {
	hash := strings.Split(c.FormValue("media"), ".")[0]
	if err := banImage(hash); err != nil { return err }
	if err := db.RemoveMedia(hash); err != nil { return err }
	return nil
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

func redirectHTTPS(next echo.HandlerFunc) echo.HandlerFunc {
	return func (c echo.Context) error {
		prefix := "/.well-known/acme-challenge/"
		if strings.HasPrefix(c.Request().RequestURI, prefix) {
			return proxyAcme(c)
		}
		parts := strings.Split(c.Request().Host, ":")
		host := parts[0]
		port := ""
		if len(parts) > 1 {
			parts = strings.Split(config.Cfg.SSL.Listener, ":")
			if len(parts) > 1 {
				port = ":" + parts[1]
			}
		}
		return c.Redirect(http.StatusFound,
			"https://" + host + port + c.Request().RequestURI)
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

func pendingMediaImage(c echo.Context) error {
	if config.Cfg.Media.PendingMime == "" {
		return c.Blob(http.StatusOK, "image/png", pendingMedia)
	}
	return c.Blob(http.StatusOK, config.Cfg.Media.PendingMime,
			config.Cfg.Media.PendingMedia)
}

func Init() error {

	sessions.Init()
	reloadRatelimits()

	if !config.Cfg.Media.InDatabase {
		os.MkdirAll(config.Cfg.Media.Path + "/thumbnail", 0700)
	}
	os.MkdirAll(config.Cfg.Media.Tmp, 0700)

	r := echo.New()
	if err := initTemplate(); err != nil { return err }

	r.Use(logger)
	r.Use(err)
	r.Use(csrf)

	r.GET("/", renderFile("index.html"))
	r.GET("/favicon.ico", notFound)
	r.GET("/robots.txt", func(c echo.Context) error {
		return c.Blob(http.StatusOK, "text/plain", robots)
	})
	r.GET("/static/favicon", func(c echo.Context) error {
		if config.Cfg.Home.FaviconMime == "" {
			return c.Blob(http.StatusOK, "image/png", favicon)
		}
		return c.Blob(http.StatusOK, config.Cfg.Home.FaviconMime,
				config.Cfg.Home.Favicon)
	})
	r.GET("/static/pending", pendingMediaImage)
	r.GET("/static/common.css", func(c echo.Context) error {
		return c.Blob(http.StatusOK, "text/css", stylesheet)
	})
	r.GET("/static/:file", func(c echo.Context) error {
		content, ok := themesContent[c.Param("file")]
		if !ok { return notFound(c) }
		serveMedia(c, content, c.Param("file"))
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
	r.GET("/disconnect/:csrf", disconnect)
	r.GET("/login", unauth(renderFile("login.html")))
	r.POST("/login", unauth(catch(loginAs, "login-error")))
	r.GET("/register", unauth(renderFile("register.html")))
	r.POST("/register", unauth(catch(readOnly(register), "register-error")))
	r.GET("/:board/cancel/:id/:csrf",
		cancel)
	r.GET("/:board/remove/:id/:csrf",
		hasRank(onPost(remove), db.RANK_ADMIN))
	r.GET("/:board/hide/:id/:csrf",
		hasRank(onPost(hide), db.RANK_MODERATOR))
	r.GET("/:board/remove_media/:id/:csrf",
		hasRank(onPost(removeMedia), db.RANK_MODERATOR))
	r.GET("/:board/ban_media/:id/:csrf",
		hasRank(onPost(banMedia), db.RANK_MODERATOR))
	r.GET("/:board/approve/:id/:csrf",
		hasRank(onPost(approveMediaFromPost), db.RANK_MODERATOR))
	r.GET("/:board/ban/:ip/:csrf", hasRank(ban, db.RANK_MODERATOR))
	if config.Cfg.Media.ApprovalQueue {
		r.GET("/approval", hasRank(
			renderFile("approval.html"), db.RANK_MODERATOR))
		r.POST("/approval/accept", hasRank(redirect(
			approveMedia, "/approval"), db.RANK_MODERATOR))
		r.POST("/approval/deny", hasRank(redirect(
			denyMedia, "/approval"), db.RANK_MODERATOR))
		r.POST("/approval/ban", hasRank(redirect(
			banPendingMedia, "/approval"), db.RANK_MODERATOR))
		r.POST("/approval/accept/all", hasRank(redirect(
			approveAll, "/approval"), db.RANK_MODERATOR))
	}
	r.GET("/dashboard", hasRank(renderDashboard, db.RANK_ADMIN))
	r.GET("/dashboard/:page", hasRank(renderDashboard, db.RANK_ADMIN))
	r.POST("/config/client/theme", func(c echo.Context) error {
		return redirect(setTheme, c.QueryParam("origin"))(c)
	})
	r.POST("/config/update", handleConfig(updateConfig, "main"))
	r.POST("/config/media/update",
		handleConfig(updateMedia, "media"))
	r.POST("/config/media/clear",
		handleConfig(clearPendingMediaImage, "media"))
	r.POST("/config/media/ban",
		handleConfig(updateMedia, "media"))
	r.POST("/config/media/ban/cancel",
		handleConfig(clearPendingMediaImage, "media"))
	r.POST("/config/ssl/update", handleConfig(updateSSL, "ssl"))
	r.POST("/config/board/create",
		handleConfig(createBoard, "board"))
	r.POST("/config/board/update/:board",
		handleConfig(updateBoard, "board"))
	r.POST("/config/board/delete/:board",
		handleConfig(deleteBoard, "board"))
	r.POST("/config/theme/create",
		handleConfig(createTheme, "theme"))
	r.POST("/config/theme/delete/:id",
		handleConfig(deleteTheme, "theme"))
	r.POST("/config/theme/update/:id",
		handleConfig(updateTheme, "theme"))
	r.POST("/config/favicon/update",
		handleConfig(updateFavicon, "favicon"))
	r.POST("/config/favicon/clear",
		handleConfig(clearFavicon, "favicon"))
	r.POST("/config/ban/create", handleConfig(addBan, "ban"))
	r.POST("/config/ban/cancel/:id", handleConfig(deleteBan, "ban"))
	r.POST("/config/account/create",
		handleConfig(addAccount, "account"))
	r.POST("/config/account/update/:id",
		handleConfig(updateAccount, "account"))
	r.POST("/config/account/delete/:id",
		handleConfig(deleteAccount, "account"))
	r.POST("/config/restart", handleConfig(restartStandard, "main"))
	r.POST("/config/acme/update", handleConfig(fetchSSL, "acme"))
	r.POST("/config/banner/create",
		handleConfig(addBanner, "banner"))
	r.POST("/config/banner/delete/:id",
		handleConfig(deleteBanner, "banner"))
	r.POST("/config/ratelimit/update",
		handleConfig(rateLimits, "rate-limit"))
	r.GET("/banner/:id", imageError(banner))
	r.GET("/.well-known/acme-challenge/:token", proxyAcme)

	if config.Cfg.Media.InDatabase {
		r.GET("/media/:hash", imageError(mediaCheck(
			func(c echo.Context) error {
				parts := strings.Split(c.Param("hash"), ".")
				data, _, err := db.GetMedia(parts[0])
				if err != nil { return err }
				serveMedia(c, data, c.Param("hash"))
				return nil
			})))
		r.GET("/media/thumbnail/:hash", imageError(mediaCheck(
			func(c echo.Context) error {
				parts := strings.Split(c.Param("hash"), ".")
				data, err := db.GetThumbnail(parts[0])
				if err != nil { return err }
				serveMedia(c, data, c.Param("hash"))
				return nil
			})))
	} else if config.Cfg.Media.ApprovalQueue {
		f := func(c echo.Context) error {
			path := c.Request().RequestURI
			path = strings.TrimPrefix(path, "/media")
			path = config.Cfg.Media.Path + path
			f, err := os.Open(path)
			if err != nil { return err }
			http.ServeContent(c.Response().Writer,
				c.Request(), c.Param("hash"),
				time.Now(), f)
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
	if !config.Cfg.SSL.Enabled {
		return r.Start(config.Cfg.Web.Listener)
	}
	if config.Cfg.SSL.RedirectToSSL {
		rdr := echo.New()
		rdr.Pre(redirectHTTPS)
		go rdr.Start(config.Cfg.Web.Listener)
	} else if !config.Cfg.SSL.DisableHTTP {
		go r.Start(config.Cfg.Web.Listener)
	}
	return r.StartTLS(config.Cfg.SSL.Listener,
		config.Cfg.SSL.Certificate, config.Cfg.SSL.Key)
}
