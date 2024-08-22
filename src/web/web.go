package web

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"strings"
	
	"IB1/db"
	"IB1/config"
)

type fErr func(c *gin.Context) error

func isBanned(c *gin.Context) error {
	_, err := loggedAs(c)
	if err == nil { return err }
	return db.IsBanned(c.RemoteIP())
}

func hasRank(f fErr, rank int) fErr {
	return func(c *gin.Context) error {
		if err := needRank(c, rank); err != nil { return err }
		return f(c)
	}
}

func err(f func(c *gin.Context) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := f(c); err != nil {
			badRequest(c, err.Error())
			return
		}
	}
}

func catch(f func(c *gin.Context) error, param string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := f(c); err != nil {
			set(c)(param, err.Error())
			c.Redirect(http.StatusFound, c.Request.RequestURI)
		}
	}
}

func unauth(f gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		_, err := loggedAs(c)
		if err == nil {
			c.Redirect(http.StatusFound, "/")
			return
		}
		f(c)
	}
}

func Init() error {

	if !config.Cfg.Media.InDatabase {
		os.MkdirAll(config.Cfg.Media.Path + "/thumbnail", 0700)
	}
	os.MkdirAll(config.Cfg.Media.Tmp, 0700)

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	if err := initTemplate(); err != nil { return err }

	r.GET("/", renderFile("index.html"))
	r.GET("/favicon.ico", func(c *gin.Context) {
		c.Data(http.StatusNotFound, "text/plain", []byte("Not Found"))
	})
	r.GET("/static/favicon", func(c *gin.Context) {
		if config.Cfg.Home.FaviconMime == "" {
			c.Data(http.StatusOK, "image/png", favicon)
			return
		}
		c.Data(http.StatusOK, config.Cfg.Home.FaviconMime,
			config.Cfg.Home.Favicon)
	})
	r.GET("/static/common.css", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/css", stylesheet)
	})
	r.GET("/static/:file", func(c *gin.Context) {
		content, ok := themesContent[c.Param("file")]
		if !ok {
			badRequest(c, "file not found")
			return
		}
		c.Writer.Header().Add("Content-Type", "text/css")
		c.Writer.Write(content)
	})
	if config.Cfg.Captcha.Enabled {
		r.GET("/captcha", captchaImage)
	}
	r.GET("/:board", err(boardIndex))
	r.GET("/:board/catalog", err(catalog))
	r.POST("/:board", catch(newThread, "new-thread-error"))
	r.GET("/:board/:thread", err(thread))
	r.POST("/:board/:thread", catch(newPost, "new-post-error"))
	r.GET("/disconnect", disconnect)
	r.GET("/login", unauth(renderFile("login.html")))
	r.POST("/login", unauth(catch(loginAs, "login-error")))
	r.GET("/:board/remove/:id", err(hasRank(remove, db.RANK_ADMIN)))
	r.GET("/:board/hide/:id", err(hasRank(hide, db.RANK_MODERATOR)))
	r.GET("/:board/ban/:ip", err(hasRank(ban, db.RANK_MODERATOR)))
	r.GET("/dashboard", err(hasRank(renderDashboard, db.RANK_ADMIN)))
	r.POST("/config/client/theme", func(c *gin.Context) {
		redirect(setTheme, c.Query("origin"))(c)
	})
	r.POST("/config/update", handleConfig(updateConfig))
	r.POST("/config/board/create", handleConfig(createBoard))
	r.POST("/config/board/update/:board", handleConfig(updateBoard))
	r.POST("/config/board/delete/:board", handleConfig(deleteBoard))
	r.POST("/config/theme/create", handleConfig(createTheme))
	r.POST("/config/theme/delete/:id", handleConfig(deleteTheme))
	r.POST("/config/theme/update/:id", handleConfig(updateTheme))
	r.POST("/config/favicon/update", handleConfig(updateFavicon))
	r.POST("/config/favicon/clear", handleConfig(clearFavicon))
	r.POST("/config/ban/create", handleConfig(addBan))
	r.POST("/config/ban/cancel/:id", handleConfig(deleteBan))

	if config.Cfg.Media.InDatabase {
		r.GET("/media/:hash", func(c *gin.Context) {
			parts := strings.Split(c.Param("hash"), ".")
			data, mime, err := db.GetMedia(parts[0])
			if err != nil {
				c.Data(http.StatusBadRequest, "text/plain",
						[]byte(err.Error()))
				return
			}
			c.Data(http.StatusOK, mime, data)
		})
		r.GET("/media/thumbnail/:hash", func(c *gin.Context) {
			parts := strings.Split(c.Param("hash"), ".")
			data, err := db.GetThumbnail(parts[0])
			if err != nil {
				c.Data(http.StatusBadRequest, "text/plain",
						[]byte(err.Error()))
				return
			}
			c.Data(http.StatusOK, "image/png", data)
		})
	} else {
		r.Static("/media", config.Cfg.Media.Path)
	}

	return r.Run(config.Cfg.Web.Listener)
}
