package web

import (
	"errors"
	"strconv"
	"os"
	"os/exec"
	"syscall"
	"time"
	"io"
	"context"
	"net"
	"net/http"
	"math/rand"
	"IB1/db"
	"IB1/config"
	"IB1/acme"

	"github.com/labstack/echo/v4"
)

var invalidForm = errors.New("invalid form")

func getPostForm(c echo.Context, param string) (string, bool) {
	v := c.Request().PostFormValue(param)
	return v, v != ""
}

func redirect(f echo.HandlerFunc, redirect string) echo.HandlerFunc {
	return func(c echo.Context) error {
		err := f(c)
		if err != nil { return err }
		c.Redirect(http.StatusFound, redirect)
		return nil
	}
}

func needRank(c echo.Context, rank int) error {
	ret := errors.New("insufficient privilege")
	var account db.Account
	account.Logged = false
	token := getCookie(c, "token")
	if token == "" { return ret }
	var err error
	account, err = db.GetAccountFromToken(token)
	if err != nil { return err }
	if account.Rank < rank { return ret }
	return nil
}

func handleConfig(f echo.HandlerFunc, param string) echo.HandlerFunc {
	return catchCustom(redirect(hasRank(f, db.RANK_ADMIN), "/dashboard"),
				param, "/dashboard")
}

func canSetConfig(c echo.Context, f echo.HandlerFunc) echo.HandlerFunc {
	if err := needRank(c, db.RANK_ADMIN); err != nil {
		return func(c echo.Context) error { return err }
	}
	return f
}

func setDefaultTheme(c echo.Context) error {
	theme, ok := getPostForm(c, "theme")
        if !ok { return invalidForm }
        _, ok = getThemesTable()[theme]
        if !ok { return errors.New("invalid theme") }
	config.Cfg.Home.Theme = theme
	return nil
}

func updateConfig(c echo.Context) error {
	requireRestart := false

	if err := setDefaultTheme(c); err != nil { return err }

	title, ok := getPostForm(c, "title")
        if !ok { return invalidForm }
	config.Cfg.Home.Title = title

	description, ok := getPostForm(c, "description")
        if !ok { return invalidForm }
	config.Cfg.Home.Description = description

	listener, ok := getPostForm(c, "listener")
        if !ok { return invalidForm }
	if !requireRestart {
		requireRestart = config.Cfg.Web.Listener != listener
	}
	config.Cfg.Web.Listener = listener

	domain, ok := getPostForm(c, "domain")
        if !ok { return invalidForm }
	config.Cfg.Web.Domain = domain

	defaultname, ok := getPostForm(c, "defaultname")
        if !ok { return invalidForm }
	config.Cfg.Post.DefaultName = defaultname

	captcha, _ := getPostForm(c, "captcha")
	config.Cfg.Captcha.Enabled = captcha == "on"

	ascii, _ := getPostForm(c, "ascii")
	config.Cfg.Post.AsciiOnly = ascii == "on"

	readonly, _ := getPostForm(c, "readonly")
	config.Cfg.Post.ReadOnly = readonly == "on"

	registration, _ := getPostForm(c, "registration")
	config.Cfg.Accounts.AllowRegistration = registration == "on"

	threadsStr, _ := getPostForm(c, "maxthreads")
	threads, err := strconv.ParseUint(threadsStr, 10, 64)
	if err != nil { return err }
	config.Cfg.Board.MaxThreads = uint(threads)

	if err := db.UpdateConfig(); err != nil { return err }
	if requireRestart { return restart(c) }
	return nil
}

func updateMedia(c echo.Context) error {
	requireRestart := false

	indb, _ := getPostForm(c, "indb")
	v := indb == "on"
	if v != config.Cfg.Media.InDatabase {
		config.Cfg.Media.InDatabase = v
		requireRestart = true
	}

	approval, _ := getPostForm(c, "approval")
	v = approval == "on"
	if v != config.Cfg.Media.ApprovalQueue {
		config.Cfg.Media.ApprovalQueue = v
		requireRestart = true
	}

	tmp, ok := getPostForm(c, "tmp")
        if !ok { return invalidForm }
	err := os.MkdirAll(config.Cfg.Media.Tmp, 0700)
	if err != nil { return err }
	config.Cfg.Media.Tmp = tmp

	path, _ := getPostForm(c, "media")
	if path == "" { path = config.Cfg.Media.Path }
	if !config.Cfg.Media.InDatabase {
		err = os.MkdirAll(path + "/thumbnail", 0700)
	}
	if err != nil { return err }
	config.Cfg.Media.Path = path

	sizeStr, _ := getPostForm(c, "maxsize")
	size, err := strconv.ParseUint(sizeStr, 10, 64)
	if err != nil { return err }
	config.Cfg.Media.MaxSize = size

	video, _ := getPostForm(c, "video")
	v = video == "on"
	if v && !config.Cfg.Media.AllowVideos {
		c := exec.Command("ffmpeg", "-version")
		if err := c.Run(); err != nil { return err }
	}
	config.Cfg.Media.AllowVideos = v

	data, mime, err := handleImage(c, "pending")
	if err == nil {
		config.Cfg.Media.PendingMedia = data
		config.Cfg.Media.PendingMime = mime
	}

	if err := db.UpdateConfig(); err != nil { return err }
	if requireRestart { return restart(c) }
	return nil
}

func clearPendingMediaImage(c echo.Context) error {
	config.Cfg.Media.PendingMime = ""
	config.Cfg.Media.PendingMedia = nil
	db.UpdateConfig()
	return nil
}

func loadFile(c echo.Context, name string) ([]byte, error) {
	file, err := c.FormFile(name)
        if err != nil { return nil, err }
	f, err := file.Open()
	if err != nil { return nil, err }
	return io.ReadAll(f)
}

func updateSSL(c echo.Context) error {

	v, _ := getPostForm(c, "enabled")
	config.Cfg.SSL.Enabled = v == "on"

	v, _ = getPostForm(c, "disable-http")
	config.Cfg.SSL.DisableHTTP = v == "on"

	v, _ = getPostForm(c, "redirect")
	config.Cfg.SSL.RedirectToSSL = v == "on"

	listener, ok := getPostForm(c, "address")
        if !ok { return invalidForm }
	config.Cfg.SSL.Listener = listener

	data, err := loadFile(c, "certificate")
        if err == nil { config.Cfg.SSL.Certificate = data }
	data, err = loadFile(c, "key")
        if err == nil { config.Cfg.SSL.Key = data }

	if err := db.UpdateConfig(); err != nil { return err }
	return restart(c)
}


func createBoard(c echo.Context) error {
	board, hasBoard := getPostForm(c, "board")
	name, hasName := getPostForm(c, "name")
        if !hasBoard || !hasName { return invalidForm }
	description, _ := getPostForm(c, "description")
	err := db.CreateBoard(board, name, description)
	if err != nil { return err }
	return db.LoadBoards()
}

func updateBoard(c echo.Context) error {
	board, hasBoard := getPostForm(c, "board")
	name, hasName := getPostForm(c, "name")
        if !hasBoard || !hasName { return invalidForm }
	enabled, _ := getPostForm(c, "enabled")
	description, _ := getPostForm(c, "description")
	boards, err := db.GetBoards()
	if err != nil { return err }
	for _, v := range boards {
		if strconv.Itoa(int(v.ID)) != c.Param("board") { continue }
		v.Name = board
		v.LongName = name
		v.Description = description
		v.Disabled = enabled != "on"
		if err := db.UpdateBoard(v); err != nil { return err }
		return db.LoadBoards()
	}
        return errors.New("invalid board")
}

func deleteBoard(c echo.Context) error {
	for i, v := range db.Boards {
		if strconv.Itoa(int(v.ID)) != c.Param("board") { continue }
		err := db.DeleteBoard(v)
		if err != nil { return err }
		delete(db.Boards, i)
		return nil
	}
        return errors.New("invalid board")
}

func createTheme(c echo.Context) error {
	file, err := c.FormFile("theme")
        if err != nil { return err }
	name, hasName := getPostForm(c, "name")
        if !hasName { return invalidForm }
	enabled, _ := getPostForm(c, "enabled")
	disabled := enabled != "on"
	data := make([]byte, file.Size)
	f, err := file.Open()
	if err != nil { return err }
	defer f.Close()
	_, err = f.Read(data)
	if err != nil { return err }
	data, err = minifyCSS(data)
	if err != nil { return err }
	err = db.AddTheme(name, string(data), disabled)
	if err != nil { return err }
	reloadThemes()
	return nil
}

func updateTheme(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil { return errors.New("invalid theme") }
	name, hasName := getPostForm(c, "name")
        if !hasName { return invalidForm }
	enabled, _ := getPostForm(c, "enabled")
	disabled := enabled != "on"
	err = db.UpdateThemeByID(id, name, disabled)
	if err != nil { return err }
	reloadThemes()
	return nil
}

func deleteTheme(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil { return errors.New("invalid theme") }
	err = db.DeleteThemeByID(id)
	if err != nil { return err }
	reloadThemes()
	return nil
}

func addBan(c echo.Context) error {
	ip, hasIP := getPostForm(c, "ip")
        if !hasIP { return invalidForm }
	expiry, hasExpiry := getPostForm(c, "expiration")
	duration := int64(3600)
        if hasExpiry {
		expiration, err := time.Parse("2006-01-02T03:04", expiry)
		if err == nil {
			duration =  expiration.Unix() - time.Now().Unix()
		}
	}
	return db.BanIP(ip, duration)
}

func deleteBan(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil { return errors.New("invalid theme") }
	return db.RemoveBan(uint(id))
}

func addAccount(c echo.Context) error {
	name := c.Request().PostFormValue("name")
	password := c.Request().PostFormValue("password")
	rank, err := db.StringToRank(c.Request().PostFormValue("rank"))
	if err != nil { return err }
	return db.CreateAccount(name, password, rank)
}

func updateAccount(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil { return errors.New("invalid user") }
	name := c.Request().PostFormValue("name")
	password := c.Request().PostFormValue("password")
	rank, err := db.StringToRank(c.Request().PostFormValue("rank"))
	if err != nil { return err }
	return db.UpdateAccount(id, name, password, rank)
}

func deleteAccount(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil { return errors.New("invalid user") }
	return db.RemoveAccount(uint(id))
}

func restart(c echo.Context) error {
	go func() {
		time.Sleep(time.Second)
		err := syscall.Exec(os.Args[0], os.Args, os.Environ())
		if err != nil {
			set(c)("restart-error",
				"Restart failed: " + err.Error())
		}
	}()
	set(c)("restart", "Restart is in progress")
	return c.Redirect(http.StatusFound, "/dashboard")
}

func fetchSSL(c echo.Context) error {
	config.Cfg.Acme.Email, _ = getPostForm(c, "email")
	v, _ := getPostForm(c, "disable-www")
	config.Cfg.Acme.DisableWWW = v == "ok"
	config.Cfg.Acme.Port = strconv.Itoa(rand.Int() % 62535 + 2048)
	crt, key, err := acme.Generate(
		config.Cfg.Web.Domain, config.Cfg.Acme.Email,
		config.Cfg.Acme.Port, !config.Cfg.Acme.DisableWWW)
	config.Cfg.Acme.Port = ""
	if err != nil { return err }
	config.Cfg.SSL.Certificate = crt
	config.Cfg.SSL.Key = key
	if err := db.UpdateConfig(); err != nil { return err }
	return restart(c)
}

func proxyAcme(c echo.Context) error {
	if config.Cfg.Acme.Port == "" { return errors.New("not found") }
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	transport := http.DefaultTransport.(*http.Transport)
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return dialer.DialContext(ctx, network,
				"127.0.0.1:" + config.Cfg.Acme.Port)
	}
	client := &http.Client{
		Transport: transport,
	}
	resp, err := client.Get(
		"http://" + c.Request().Host + c.Request().RequestURI)
	if err != nil { return err }
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil { return err }
	return c.Blob(resp.StatusCode, resp.Header.Get("Content-Type"), data)
}
