package web

import (
	"github.com/labstack/echo/v4"
	"net/http"
	"hash/fnv"
	"bytes"
	"os"
	"os/signal"
	"syscall"
	"io/fs"
	"strings"
	"time"
	"strconv"
	"log"
	
	"IB1/db"
	"IB1/dnsbl"
	"IB1/config"
)

func serveMedia(c echo.Context, data []byte, name string) {
	r := bytes.NewReader(data)
	http.ServeContent(c.Response().Writer, c.Request(), name, time.Now(), r)
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
		if !config.Cfg.Media.ApprovalQueue { return f(c) }
		user, err := loggedAs(c)
		if err == nil && user.Can(db.VIEW_PENDING_MEDIA) == nil {
			return f(c)
		}
		hash := strings.Split(c.Param("hash"), ".")[0]
		media, err := db.GetMedia(hash)
		if err != nil { return err }
		if media.Approved { return f(c) }
		return pendingMediaImage(c)
	}
}

func thumbnailCheck(f echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		hash := strings.Split(c.Param("hash"), ".")[0]
		media, err := db.GetMedia(hash)
		if err != nil { return err }
		if media.HideThumbnail { return spoilerImage(c) }
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
			if c.Response().Status == 200 {
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
		blacklist := ""
		if err := isBlacklisted(c); err != nil {
			blacklist = "(blacklisted)"
		}
		log.Println(
			"[" + id + "][" + ip + blacklist + "][" +
			r.Method + "]" + name,
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
	board, err := db.GetBoard(c.Param("board"))
	if err != nil {
		return db.IsBanned(clientIP(c), board.ID)
	}
	return db.IsBanned(clientIP(c), 0)
}

func hasPrivilege(f echo.HandlerFunc, privilege db.Privilege) echo.HandlerFunc {
	return func(c echo.Context) error {
		if err := needPrivilege(c, privilege); err != nil { return err }
		return f(c)
	}
}

func hasBoardPrivilege(f echo.HandlerFunc,
		privilege db.MemberPrivilege) echo.HandlerFunc {
	return func(c echo.Context) error {
		if needPrivilege(c, privilege.Generic()) != nil {
			user, err := loggedAs(c)
			if err != nil { return err }
			board, err := db.GetBoard(c.Param("board"))
			if err != nil { return err }
			err = user.CanAsMember(board, privilege)
			if err != nil { return err }
		}
		return f(c)
	}
}

func canView(c echo.Context, board db.Board) error {
	if !board.Private { return nil }
	if board.Disabled { return invalidRequest }
	acc, err := loggedAs(c)
	if err == nil {
		if board.OwnerID != nil && acc.ID == *board.OwnerID {
			return nil
		}
		_, err := board.GetMember(acc)
		if err == nil { return nil }
	}
	priv := db.USE_PRIVATE
	if c.Request().Method == "GET" {
		priv = db.VIEW_PRIVATE
	}
	err = needPrivilege(c, priv)
	if err != nil { return err }
	return nil
}

func privateBoard(f echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		name := c.Param("board")
		if name == "" { return f(c) }
		board, err := db.GetBoard(name)
		if err != nil { return err }
		if err := canView(c, board); err != nil { return err }
		return f(c)
	}
}

func isReadOnly(c echo.Context) bool {
	if c.Request().Method != "GET" {
		return false
	}
	for _, v := range c.ParamNames() {
		if v != "csrf" { continue }
		return false
	}
	return true
}

func isBlacklisted(c echo.Context) error {
	if strings.HasPrefix(c.Request().RequestURI, "/static/") {
		return nil
	}
	return dnsbl.IsListed(clientIP(c), isReadOnly(c))
}


func blacklistCheck(f echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if err := isBlacklisted(c); err != nil {
			return err
		}
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

func auth(f echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		_, err := loggedAs(c)
		if err != nil {
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

func spoilerImage(c echo.Context) error {
	if config.Cfg.Media.SpoilerMime == "" {
		return c.Blob(http.StatusOK, "image/png", spoiler)
	}
	return c.Blob(http.StatusOK, config.Cfg.Media.SpoilerMime,
			config.Cfg.Media.Spoiler)
}

func Init() error {

	dnsbl.Init()
	sessions.Init()
	captchaInit()
	reloadRatelimits()
	if err := db.LoadSessions(&sessions); err != nil {
		return err
	}
	go clearSession()

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		db.SaveSessions(&sessions)
		os.Exit(1)
	}()

	if !config.Cfg.Media.InDatabase {
		os.MkdirAll(config.Cfg.Media.Path + "/thumbnail", 0700)
	}
	os.MkdirAll(config.Cfg.Media.Tmp, 0700)

	r := echo.New()
	if err := initTemplate(); err != nil { return err }

	r.Use(logger)
	r.Use(err)
	r.Use(csrf)
	r.Use(blacklistCheck)
	r.Use(privateBoard)

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
	r.GET("/static/spoiler", spoilerImage)
	r.GET("/static/common.css", func(c echo.Context) error {
		return c.Blob(http.StatusOK, "text/css", stylesheet)
	})
	r.GET("/static/:file", func(c echo.Context) error {
		content, ok := themesContent[c.Param("file")]
		if !ok { return notFound(c) }
		serveMedia(c, content, c.Param("file"))
		return nil
	})
	r.GET("/static/:file", func(c echo.Context) error {
		sub, err := fs.Sub(static, "static")
		if err != nil { return err }
		http.ServeFileFS(c.Response().Writer, c.Request(),
				sub, c.Param("file"))
		return nil
	})
	r.GET("/static/flags/:file", func(c echo.Context) error {
		sub, err := fs.Sub(flags, "static/flags")
		if err != nil { return err }
		http.ServeFileFS(c.Response().Writer, c.Request(),
				sub, c.Param("file"))
		return nil
	})
	if config.Cfg.Captcha.Enabled {
		r.GET("/captcha", captchaImage)
	}
	r.GET("/:board", boardIndex)
	r.GET("/:board/catalog", catalog)
	r.POST("/:board", catch(readOnly(hasBoardPrivilege(
		newThread, db.CREATE_THREAD.Member())), "new-thread-error"))
	r.GET("/:board/:thread", thread)
	r.POST("/:board/:thread", catch(readOnly(hasBoardPrivilege(
		newPost, db.CREATE_POST.Member())), "new-post-error"))
	r.GET("/disconnect/:csrf", disconnect)
	r.GET("/login", unauth(renderFile("login.html")))
	r.POST("/login", unauth(catch(loginAs, "login-error")))
	r.GET("/register", unauth(renderFile("register.html")))
	r.POST("/register", unauth(catch(readOnly(register), "register-error")))
	r.GET("/:board/cancel/:id/:csrf", cancel)
	r.GET("/:board/remove/:id/:csrf",
		hasBoardPrivilege(onPost(remove), db.REMOVE_POST.Member()))
	r.GET("/:board/hide/:id/:csrf",
		hasBoardPrivilege(onPost(hide), db.HIDE_POST.Member()))
	r.GET("/:board/pin/:id/:csrf",
		hasBoardPrivilege(onPost(pin), db.PIN_THREAD.Member()))
	r.GET("/:board/remove_media/:id/:csrf", hasBoardPrivilege(
		onPost(removeMedia), db.REMOVE_MEDIA.Member()))
	r.GET("/:board/ban_media/:id/:csrf",
		hasPrivilege(onPost(banMedia), db.BAN_MEDIA))
	r.GET("/:board/approve/:id/:csrf", hasBoardPrivilege(
		onPost(approveMediaFromPost), db.APPROVE_MEDIA.Member()))
	r.GET("/:board/ban/:ip/:csrf",
		hasBoardPrivilege(ban, db.BAN_USER.Member()))
	if config.Cfg.Media.ApprovalQueue {
		r.GET("/approval", hasPrivilege(
			renderFile("approval.html"), db.APPROVE_MEDIA))
		r.POST("/approval/accept", hasPrivilege(redirect(
			approveMedia, "/approval"), db.APPROVE_MEDIA))
		r.POST("/approval/deny", hasPrivilege(redirect(
			denyMedia, "/approval"), db.APPROVE_MEDIA))
		r.POST("/approval/ban", hasPrivilege(redirect(
			banPendingMedia, "/approval"), db.APPROVE_MEDIA))
		r.POST("/approval/accept/all", hasPrivilege(redirect(
			approveAll, "/approval"), db.APPROVE_MEDIA))
	}
	r.GET("/boards", redirect(renderBoards, "/"))
	r.POST("/boards/create", redirect(
		hasPrivilege(createBoard, db.CREATE_BOARD), "/boards"))
	r.POST("/boards/:id/update",
		redirect(asOwner(updateOwnedBoard), "/boards"))
	r.POST("/boards/:id/add", redirect(asOwner(addMember), "/boards"))
	r.POST("/boards/:id/remove",
		redirect(asOwner(removeMember), "/boards"))
	r.POST("/boards/:id/delete",
		redirect(asOwner(func(_ db.Board, c echo.Context)error{
			return deleteBoard(c)
		}), "/boards"))
	r.POST("/boards/:id/member",
		redirect(asOwner(updateMember), "/boards"))
	r.GET("/dashboard",
		hasPrivilege(renderDashboard, db.ADMINISTRATION))
	r.GET("/dashboard/:page",
		hasPrivilege(renderDashboard, db.ADMINISTRATION))
	r.POST("/config/client/theme", func(c echo.Context) error {
		return redirect(setTheme, c.QueryParam("origin"))(c)
	})
	r.POST("/config/update", handleConfig(updateConfig, "main"))
	r.POST("/config/geo/update", handleConfig(updateGeoIP, "main"))
	r.POST("/config/media/update",
		handleConfig(updateMedia, "media"))
	r.POST("/config/media/pending/clear",
		handleConfig(clearPendingMediaImage, "media"))
	r.POST("/config/media/spoiler/clear",
		handleConfig(clearSpoilerImage, "media"))
	r.POST("/config/media/ban",
		handleConfig(addBannedHash, "media"))
	r.POST("/config/media/ban/cancel",
		handleConfig(removeBannedHash, "media"))
	r.POST("/config/ssl/update", handleConfig(updateSSL, "ssl"))
	r.POST("/config/board/create",
		handleConfig(createBoard, "board"))
	r.POST("/config/board/update/:id",
		handleConfig(updateBoard, "board"))
	r.POST("/config/board/delete/:id",
		handleConfig(deleteBoard, "board"))

	r.POST("/config/theme/create", handleConfig(createTheme, "theme"))
	r.POST("/config/theme/delete/:id", handleConfig(deleteTheme, "theme"))
	r.POST("/config/theme/update/:id", handleConfig(updateTheme, "theme"))

	r.POST("/config/wordfilter/create",
		handleConfig(createWordfilter, "wordfilter"))
	r.POST("/config/wordfilter/delete/:id",
		handleConfig(deleteWordfilter, "wordfilter"))
	r.POST("/config/wordfilter/update/:id",
		handleConfig(updateWordfilter, "wordfilter"))

	r.POST("/config/blacklist/create",
		handleConfig(createBlacklist, "blacklist"))
	r.POST("/config/blacklist/delete/:id",
		handleConfig(deleteBlacklist, "blacklist"))
	r.POST("/config/blacklist/update/:id",
		handleConfig(updateBlacklist, "blacklist"))

	r.POST("/config/rank/create", handleConfig(createRank, "rank"))
	r.POST("/config/rank/delete/:id", handleConfig(deleteRank, "rank"))
	r.POST("/config/rank/update/:id", handleConfig(updateRank, "rank"))

	r.POST("/config/member/rank/create",
		handleConfig(createMemberRank, "rank"))
	r.POST("/config/member/rank/delete/:id",
		handleConfig(deleteMemberRank, "rank"))
	r.POST("/config/member/rank/update/:id",
		handleConfig(updateMemberRank, "rank"))

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
				data, _, err := db.GetMediaData(parts[0])
				if err != nil { return err }
				serveMedia(c, data, c.Param("hash"))
				return nil
			})))
		r.GET("/media/thumbnail/:hash", imageError(
			thumbnailCheck(mediaCheck(func(c echo.Context) error {
				parts := strings.Split(c.Param("hash"), ".")
				data, err := db.GetThumbnail(parts[0])
				if err != nil { return err }
				serveMedia(c, data, c.Param("hash"))
				return nil
			}))))
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
		r.GET("/media/thumbnail/:hash",
			imageError(thumbnailCheck(mediaCheck(f))))
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
