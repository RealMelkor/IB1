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
	"net/url"
	"net/http"
	"math/rand"
	"IB1/db"
	"IB1/config"
	"IB1/acme"

	"github.com/labstack/echo/v4"
)

var invalidForm = errors.New("invalid form")
var invalidID = errors.New("invalid id")
var invalidRequest = errors.New("invalid request")

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

func needPrivilege(c echo.Context, privilege db.Privilege) error {
	token := getCookie(c, "token")
	if token != "" {
		account, err := db.GetAccountFromToken(token)
		if err == nil {
			return account.Can(privilege)
		}
	}
	v, err := db.AsUnauthenticated(privilege)
	if err != nil { return err }
	if !v { return errors.New("insufficient privilege") }
	return nil
}

func handleConfig(f echo.HandlerFunc, param string) echo.HandlerFunc {
	dst := "/dashboard/" + param
	return catchCustom(redirect(hasPrivilege(f, db.ADMINISTRATION), dst),
			param + "-error", dst)
}

func canSetConfig(c echo.Context, f echo.HandlerFunc) echo.HandlerFunc {
	if err := needPrivilege(c, db.ADMINISTRATION); err != nil {
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

	config.Cfg.Accounts.DefaultRank, _ = getPostForm(c, "defaultrank")

	threadsStr, _ := getPostForm(c, "maxthreads")
	threads, err := strconv.ParseUint(threadsStr, 10, 64)
	if err != nil { return err }
	config.Cfg.Board.MaxThreads = uint(threads)

	if err := db.UpdateConfig(); err != nil { return err }
	if requireRestart { return restart(c, "main") }
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

	thresholdStr, _ := getPostForm(c, "threshold")
	threshold, err := strconv.Atoi(thresholdStr)
	if err != nil { return err }
	config.Cfg.Media.ImageThreshold = threshold

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

	data, mime, err = handleImage(c, "spoiler")
	if err == nil {
		config.Cfg.Media.Spoiler = data
		config.Cfg.Media.SpoilerMime = mime
	}

	if err := db.UpdateConfig(); err != nil { return err }
	if requireRestart { return restart(c, "media") }
	return nil
}

func clearPendingMediaImage(echo.Context) error {
	config.Cfg.Media.PendingMime = ""
	config.Cfg.Media.PendingMedia = nil
	db.UpdateConfig()
	return nil
}

func clearSpoilerImage(echo.Context) error {
	config.Cfg.Media.SpoilerMime = ""
	config.Cfg.Media.Spoiler = nil
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
	return restart(c, "ssl")
}


func createBoard(c echo.Context) error {
	board, hasBoard := getPostForm(c, "board")
	name, hasName := getPostForm(c, "name")
        if !hasBoard || !hasName { return invalidForm }
	description, _ := getPostForm(c, "description")
	acc, err := loggedAs(c)
	if err != nil { return err }
	err = db.CreateBoard(board, name, description, acc.ID)
	if err != nil { return err }
	return db.LoadBoards()
}

func updateBoard(c echo.Context) error {
	board, hasBoard := getPostForm(c, "board")
	name, hasName := getPostForm(c, "name")
        if !hasBoard || !hasName { return invalidForm }
	enabled, _ := getPostForm(c, "enabled")
	description, _ := getPostForm(c, "description")
	countryFlag, _ := getPostForm(c, "country-flag")
	posterID, _ := getPostForm(c, "poster-id")
	readonly, _ := getPostForm(c, "read-only")
	private, _ := getPostForm(c, "private")
	owner, _ := getPostForm(c, "owner")
	boards, err := db.GetBoards()
	if err != nil { return err }
	for _, v := range boards {
		if strconv.Itoa(int(v.ID)) != c.Param("id") { continue }
		v.Name = board
		v.LongName = name
		v.Description = description
		v.Disabled = enabled != "on"
		v.CountryFlag = countryFlag == "on"
		v.PosterID = posterID == "on"
		v.ReadOnly = readonly == "on"
		v.Private = private == "on"
		if owner != "" {
			account, err := db.GetAccount(owner)
			if err != nil { return err }
			v.OwnerID = &account.ID
		} else {
			v.OwnerID = nil
		}
		if err := db.UpdateBoard(v); err != nil { return err }
		return db.LoadBoards()
	}
        return errors.New("invalid board")
}

func deleteBoard(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil { return err }
	for i, v := range db.Boards {
		if v.ID != uint(id) { continue }
		delete(db.Boards, i)
	}
	v := db.Board{}
	v.ID = uint(id)
	return db.DeleteBoard(v)
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
	err = db.Theme{}.Add(db.Theme{
		Name: name, Content: string(data), Disabled: disabled,
	})
	if err != nil { return err }
	reloadThemes()
	return nil
}

func updateTheme(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil { return invalidID }
	name, hasName := getPostForm(c, "name")
        if !hasName { return invalidForm }
	enabled, _ := getPostForm(c, "enabled")
	disabled := enabled != "on"
	err = db.Theme{}.Update(id, db.Theme{
		Name: name, Disabled: disabled,
	})
	if err != nil { return err }
	reloadThemes()
	return nil
}

func deleteTheme(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil { return invalidID }
	err = db.Theme{}.RemoveID(id, db.Theme{})
	if err != nil { return err }
	reloadThemes()
	return nil
}

func createBlacklist(c echo.Context) error {
	host, hasHost:= getPostForm(c, "host")
        if !hasHost { return invalidForm }
	v, err := url.Parse("https://" + host + "/")
	if err != nil { return err }

	enabled, _ := getPostForm(c, "enabled")
	disabled := enabled != "on"
	return db.Blacklist{}.Add(db.Blacklist{
		Disabled: disabled,
		Host: v.Hostname(),
	})
}

func deleteBlacklist(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil { return invalidID }
	return db.Blacklist{}.RemoveID(id, db.Blacklist{})
}

func addBan(c echo.Context) error {
	ip, hasIP := getPostForm(c, "ip")
        if !hasIP { return invalidForm }
	board, _ := getPostForm(c, "board")
	boardID, err := strconv.Atoi(board)
	if err != nil { return err }
	expiry, hasExpiry := getPostForm(c, "expiration")
	duration := int64(3600)
        if hasExpiry {
		expiration, err := time.Parse("2006-01-02T03:04", expiry)
		if err == nil {
			duration =  expiration.Unix() - time.Now().Unix()
		}
	}
	return db.BanIP(ip, duration, uint(boardID))
}

func deleteBan(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil { return invalidID }
	return db.RemoveBan(uint(id))
}

func asOwner(f func(db.Board, echo.Context)error) echo.HandlerFunc {
	return catchCustom(func(c echo.Context) error {
		account, err := loggedAs(c)
		if err != nil { return err }
		boards, err := account.GetBoards()
		if err != nil { return err }
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil { return err }
		for _, v := range boards {
			if v.ID == uint(id) {
				return f(v, c)
			}
		}
		return invalidID
	}, "boards-error", "/boards")
}

func updateOwnedBoard(_ db.Board, c echo.Context) error {
	acc, err := loggedAs(c)
	if err != nil { return err }
	v, _ := getPostForm(c, "owner")
	if acc.Name != v { return invalidForm }
	return updateBoard(c)
}

func addMember(v db.Board, c echo.Context) error {
	name := c.Request().PostFormValue("name")
	rank := c.Request().PostFormValue("rank")
	return v.AddMember(name, rank)
}

func removeMember(v db.Board, c echo.Context) error {
	return v.RemoveMember(c.Request().PostFormValue("name"))
}

func updateMember(v db.Board, c echo.Context) error {
	name := c.Request().PostFormValue("name")
	rank := c.Request().PostFormValue("rank")
	return v.UpdateMember(name, rank)
}

func addAccount(c echo.Context) error {
	name := c.Request().PostFormValue("name")
	password := c.Request().PostFormValue("password")
	rank := c.Request().PostFormValue("rank")
	return db.CreateAccount(name, password, rank, false)
}

func updateAccount(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil { return invalidID }
	name := c.Request().PostFormValue("name")
	password := c.Request().PostFormValue("password")
	rank := c.Request().PostFormValue("rank")
	return db.UpdateAccount(id, name, password, rank)
}

func deleteAccount(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil { return invalidID }
	return db.RemoveAccount(uint(id))
}

func restartStandard(c echo.Context) error {
	return restart(c, "main")
}

func restart(c echo.Context, redirect string) error {
	go func() {
		time.Sleep(time.Second)
		err := syscall.Exec(os.Args[0], os.Args, os.Environ())
		if err != nil {
			set(c)("restart-error",
				"Restart failed: " + err.Error())
		}
	}()
	set(c)("restart", "Restart is in progress")
	return c.Redirect(http.StatusFound, "/dashboard/" + redirect)
}

func updateGeoIP(c echo.Context) error {
	err := db.UpdateZones(db.ZonesURL)
	if err != nil { return err }
	set(c)("info", "Zones updated succesfully")
	return db.LoadCountries()
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
	return restart(c, "ssl")
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

func getInt(c echo.Context, param string) (int, error) {
	str, ok := getPostForm(c, param)
	if !ok { return 0, errors.New("missing parameter") }
	v, err := strconv.Atoi(str)
	if err != nil { return 0, err }
	if v < 0 { return 0, errors.New("invalid value") }
	return v, nil
}

func rateLimits(c echo.Context) error {
	var err error
	tmp := config.Cfg.RateLimit

	tmp.Login.MaxAttempts, err = getInt(c, "login-attempts")
	if err != nil { return err }
	tmp.Login.Timeout , err = getInt(c, "login-timeout")
	if err != nil { return err }

	tmp.Account.MaxAttempts, err = getInt(c, "account-attempts")
	if err != nil { return err }
	tmp.Account.Timeout , err = getInt(c, "account-timeout")
	if err != nil { return err }

	tmp.Registration.MaxAttempts, err = getInt(c, "register-attempts")
	if err != nil { return err }
	tmp.Registration.Timeout , err = getInt(c, "register-timeout")
	if err != nil { return err }

	tmp.Post.MaxAttempts, err = getInt(c, "post-attempts")
	if err != nil { return err }
	tmp.Post.Timeout , err = getInt(c, "post-timeout")
	if err != nil { return err }

	tmp.Thread.MaxAttempts, err = getInt(c, "thread-attempts")
	if err != nil { return err }
	tmp.Thread.Timeout , err = getInt(c, "thread-timeout")
	if err != nil { return err }

	config.Cfg.RateLimit = tmp
	if err := db.UpdateConfig(); err != nil { return err }
	reloadRatelimits()
	return nil
}
