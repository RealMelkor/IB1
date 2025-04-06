package web

import (
	"embed"
	"html/template"
	"net/http"
	"strings"
	"strconv"
	"io"
	"crypto/rand"
	"math/big"
	"fmt"
	"hash/fnv"
	"encoding/base32"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/labstack/echo/v4"
	"github.com/gabriel-vasile/mimetype"

	"IB1/db"
	"IB1/config"
	"IB1/util"
)

//go:embed html/*.html
var templatesFS embed.FS

//go:embed static/*
var static embed.FS

//go:embed static/flags/*
var flags embed.FS

//go:embed static/robots.txt
var robots []byte

//go:embed static/favicon.png
var favicon []byte

//go:embed static/pending.png
var pendingMedia []byte

//go:embed static/spoiler.png
var spoiler []byte

//go:embed static/error.png
var mediaError []byte

var templates *template.Template

func isLogged(c echo.Context) bool {
	_, err := loggedAs(c)
	return err == nil
}

func render(_template string, data any, c echo.Context) error {
	c.Response().Writer.WriteHeader(http.StatusOK)
	c.Response().Writer.Header().Add(
		"Content-Type", "text/html; charset=utf-8")
	w := minifyHTML(c.Response().Writer)
	defer w.Close()
	funcs := template.FuncMap{
		"get": get(c),
		"once": once(c),
		"set": set(c),
		"has": has(c),
		"param": func(param string) any { return c.Param(param) },
		"render": func(template string, v any) error {
			return templates.Lookup(template).Execute(w, v)
		},
		"session": func() string { return getCookie(c, "id") },
		"isLogged": func() bool { return isLogged(c) },
		"can": func(priv string) bool {
			acc, err := loggedAs(c)
			if err != nil {
				v, err := db.UnauthenticatedCan(priv)
				return err == nil && v
			}
			return acc.HasPrivilege(priv) == nil
		},
		"memberCan": func(priv string) bool {
			acc, err := loggedAs(c)
			if err != nil {
				v, err := db.UnauthenticatedCan(priv)
				return err == nil && v
			}
			board, err := db.GetBoard(c.Param("board"))
			if err != nil { return false }
			return acc.CanAsMember(board,
				db.GetMemberPrivilege(priv)) == nil
		},
		"isSelf": func(acc db.Account) bool {
			self, err := loggedAs(c)
			if err != nil { return false }
			return self.ID == acc.ID
		},
		"self": func() db.Account {
			self, err := loggedAs(c)
			if err != nil { return db.Account{} }
			return self
		},
		"canView": func(board db.Board) bool {
			return canView(c, board) == nil
		},
	}
	err := templates.Funcs(funcs).Lookup("header").Execute(w, header(c))
	if err != nil { return err }
	err = templates.Funcs(funcs).Lookup(_template).Execute(w, data)
	if err != nil { return err }
	err = templates.Lookup("footer").Execute(w, header(c))
	if err != nil { return err }
	return nil
}

func isMedia(media string, mediaType db.MediaType) bool {
	parts := strings.Split(media, ".")
	if len(parts) < 2 { return false }
	ext := parts[len(parts) - 1]
	v, ok := extensions["." + ext]
	if !ok { return false }
	return mediaType == v
}

func initTemplate() error {
	var err error
	funcs := template.FuncMap{
		"boards": func() []db.Board {
			boards, err := db.GetBoards()
			if err != nil { return nil }
			return boards
		},
		"isCaptchaEnabled": func() bool {
			return	config.Cfg.Captcha.Enabled
		},
		"config": func() config.Config { return config.Cfg },
		"isLogged": func() bool { return false },
		"can": func(string) bool { return false },
		"memberCan": func(string) bool { return false },
		"set": func(string, string) string {return ""},
		"get": func(string) string {return ""},
		"once": func(string) string {return ""},
		"has": func(string) bool {return false},
		"param": func(string) any { return "" },
		"render": func(string, any) error { return nil },
		"session": func() string {return ""},
		"csrf": func() string { return "" },
		"hasRank": func(string) bool {return false},
		"isSelf": func(db.Account) bool {return false},
		"self": func() db.Account {return db.Account{}} ,
		"capitalize": func(s string) string {
			if s == "" { return "" }
			return strings.ToUpper(s[0:1]) + s[1:]
		},
		"pendingMedia": func() string {
			hash, mime, err := db.GetPendingApproval()
			if err != nil { return "" }
			m := mimetype.Lookup(mime)
			if m == nil { return "" }
			return hash + m.Extension()
		},
		"isPending": func(media string) bool {
			v, err := db.GetMedia(media)
			return config.Cfg.Media.ApprovalQueue &&
					err == nil && !v.Approved
		},
		"hasUnapproved": func() bool {
			return db.HasUnapproved()
		},
		"thumbnail": func(media string) string {
			return strings.Split(media, ".")[0] + ".png"
		},
		"isPicture": func(media string) bool {
			return isMedia(media, db.MEDIA_PICTURE)
		},
		"isVideo": func(media string) bool {
			return isMedia(media, db.MEDIA_VIDEO)
		},
		"extension": func(path string) string {
			parts := strings.Split(path, ".")
			if len(parts) < 1 { return "" }
			return parts[len(parts) - 1]
		},
		"banners": func() []uint {
			v, _ := db.GetAllBanners()
			return v
		},
		"banner": func() uint {
			v, _ := db.GetAllBanners()
			i, _ := rand.Int(rand.Reader, big.NewInt(int64(len(v))))
			return v[i.Int64()]
		},
		"arr": func(args ...any) []any {
			return args
		},
		"ranks": func() []string {
			ranks, err := db.GetRanks()
			if err != nil { return nil }
			res := []string{}
			for _, v := range ranks {
				res = append(res, v.Name)
			}
			return res
		},
		"memberRanks": func() []string {
			ranks, err := db.GetMemberRanks()
			if err != nil { return nil }
			res := []string{}
			for _, v := range ranks {
				res = append(res, v.Name)
			}
			return res
		},
		"idColor": func(in string) string {
			h := fnv.New32()
			sum := h.Sum([]byte(in))
			sum[sum[3] % 3] |= 0xA0
			return fmt.Sprintf("#%02X%02X%02X",
				sum[0], sum[1], sum[2])
		},
		"country": func(code string) string {
			return util.CountryName(code)
		},
		"randID": func() string {
			var buf [8]byte
			rand.Read(buf[:])
			return base32.StdEncoding.WithPadding(base32.NoPadding).
				EncodeToString(buf[:])
		},
		"canView": func(db.Board) bool {
			return false
		},
	}
	templates, err = template.New("frontend").Funcs(funcs).
				ParseFS(templatesFS, "html/*.html")
	if err != nil { return err }
	if err := minifyStylesheet(); err != nil { return err }
	return nil
}

func header(c echo.Context) any {
	boards, err := db.GetBoards()
	if err != nil { return nil }
	account, err := loggedAs(c)
	logged := err == nil
	theme := getTheme(c)
	data := struct {
		Config	config.Config
		Url	string
		Theme	string
		Themes	[]string
		Logged	bool
		Account	db.Account
		Boards	[]db.Board
	}{
		config.Cfg,
		c.Request().RequestURI,
		theme,
		getThemes(),
		logged,
		account,
		boards,
	}
	return data
}

func renderBoards(c echo.Context) error {
	acc, err := loggedAs(c)
	if err != nil {
		return invalidRequest
	}
	if v, _ := acc.GetBoards(); len(v) < 1 {
		return invalidRequest
	}
	render("boards.html", nil, c)
	return nil
}

func renderDashboard(c echo.Context) error {
	boards, err := db.GetBoards()
	if err != nil { return err }
	accounts, err := db.GetAccounts()
	if err != nil { return err }
	themes, err := db.Theme{}.GetAll()
	if err != nil { return err }
	ranks, err := db.GetRanks()
	if err != nil { return err }
	memberRanks, err := db.GetMemberRanks()
	if err != nil { return err }
	wordfilters, err := db.Wordfilter{}.GetAll()
	if err != nil { return err }
	bannedImages, err := db.GetBannedImages()
	if err != nil { return err }
	data := struct {
		Accounts	[]db.Account
		Boards		[]db.Board
		Config		config.Config
		Theme		string
		Themes		[]string
		Privileges	[]string
		MemberPrivileges[]string
		Bans		[]db.Ban
		BannedImages	[]db.BannedImage
		UserThemes	[]db.Theme
		Wordfilters	[]db.Wordfilter
		Ranks		[]db.Rank
		MemberRanks	[]db.MemberRank
		Header		any
	}{
		Accounts: accounts,
		Boards: boards,
		Config: config.Cfg,
		Bans:	db.BanList,
		BannedImages: bannedImages,
		Themes: getThemes(),
		UserThemes: themes,
		Wordfilters: wordfilters,
		Ranks: ranks,
		MemberRanks: memberRanks,
		Privileges: db.GetPrivileges(),
		MemberPrivileges: db.GetMemberPrivileges(),
		Header: header(c),
	}
	return render("admin.html", data, c)
}

func removeDuplicateInt(intSlice []int) []int {
	allKeys := make(map[int]bool)
	list := []int{}
	for _, item := range intSlice {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}

func addLinks(content string, thread uint) (string, []int) {
	const quote = "&gt;&gt;"
	var refs []int
	for i := strings.Index(content, quote); i >= 0 &&
			i + len(quote) < len(content); {

		index := strings.Index(content[i:], quote)
		if index < 0 { break }
		i += index + len(quote)

		j := i
		length := len(content)
		for ; j < length; j++ {
			if (content[j] < '0' || content[j] > '9') { break }
		}
		if j < length && content[j] != ' ' && content[j] != '\n' &&
				content[j] == '\t' {
			continue
		}
		number := content[i:j]
		n, err := strconv.Atoi(number)
		if err != nil { continue }
		if _, err := db.GetPost(thread, n); err != nil { continue }
		refs = append(refs, n)
		str := "<a class=\"l-" + number + "\" href=\"#" + number +
			"\">&gt;&gt;" + number + "</a>"
		content = content[:i - len(quote)] + str + content[j:]
		i += len(str) - len(quote)
	}

	return content, removeDuplicateInt(refs)
}

func addGreentext(content string) string {
	const br = "<br>"
	strings.ReplaceAll(content, "\r", "")
	length := len(content)
	next := 0
	for i := 0; i >= 0 && i < length; i = next {
		next = strings.Index(content[i:], br)
		if next == -1 { next = length } else { next += i + len(br) }
		if strings.Index(content[i:next], "&gt;&gt;") == 0 { continue }
		if strings.Index(content[i:next], "&gt;") != 0 { continue }
		line := "<span class=\"green-text\">" +
				content[i:next] + "</span>"
		content = content[:i] + line + content[next:]
		length = len(content)
		next = i + len(line)
	}
	return content
}

func asciiOnly(s string) string {
	i := 0
	res := make([]byte, len(s))
	for _, c := range s {
		if c == '\t' || c == '\n' || (c >= ' ' && c < 127) {
			res[i] = byte(c)
			i++
		}
	}
	if i == 0 { return "" }
	return string(res[:i])
}

func parseContent(content string, thread uint) (template.HTML, []int) {
	if config.Cfg.Post.AsciiOnly {
		content = asciiOnly(content)
	}
	content = template.HTMLEscapeString(content)
	content = strings.Replace(content, "\n", "<br>", -1)
	content, refs := addLinks(content, thread)
	content = addGreentext(content)
	return template.HTML(content), refs
}

func minifyCSS(in []byte) ([]byte, error) {
	m := minify.New()
	m.AddFunc("text/css", css.Minify)
	res, err := m.Bytes("text/css", in)
	if err != nil { return nil, err }
	return res, nil
}

func minifyHTML(w io.Writer) io.WriteCloser {
	m := minify.New()
	m.AddFunc("text/html", html.Minify)
	m.AddFunc("text/css", css.Minify)
	return m.Writer("text/html", w)
}

var stylesheet []byte = nil
func minifyStylesheet() error {
	m := minify.New()
	m.AddFunc("text/css", css.Minify)
	data, err := static.ReadFile("static/common.css")
	if err != nil { return err }
	res, err := m.Bytes("text/css", data)
	if err != nil { return err }
	stylesheet = res
	return nil
}
