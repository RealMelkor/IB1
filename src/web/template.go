package web

import (
	"embed"
	"html/template"
	"net/http"
	"strings"
	"strconv"
	"io"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/labstack/echo/v4"

	"IB1/db"
	"IB1/config"
)

//go:embed html/*.html
var templatesFS embed.FS

//go:embed static/*
var static embed.FS

//go:embed static/favicon.png
var favicon []byte

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
		"session": func() string { return getCookie(c, "id") },
		"isLogged": func() bool { return isLogged(c) },
		"hasRank": func(rank string) bool {
			acc, err := loggedAs(c)
			if err != nil { return false }
			return acc.HasRank(rank)
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
	}
	err := templates.Lookup("header").Execute(w, header(c))
	if err != nil { return err }
	err = templates.Funcs(funcs).Lookup(_template).Execute(w, data)
	if err != nil { return err }
	err = templates.Lookup("footer").Execute(w, header(c))
	if err != nil { return err }
	return nil
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
		"set": func(string, string) string {return ""},
		"get": func(string) string {return ""},
		"once": func(string) string {return ""},
		"has": func(string) bool {return false},
		"session": func() string {return ""},
		"rank": func(rank string) int {
			i, _ := db.StringToRank(rank)
			return i
		},
		"rankToString": func(rank int) string {
			s, _ := db.RankToString(rank)
			return s
		},
		"hasRank": func(string) bool {return false},
		"isSelf": func(db.Account) bool {return false},
		"self": func() db.Account {return db.Account{}} ,
		"ranks": func() []string {
			return db.Ranks()
		},
		"capitalize": func(s string) string {
			if s == "" { return "" }
			return strings.ToUpper(s[0:1]) + s[1:]
		},
	}
	templates, err = template.New("gmi").Funcs(funcs).
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

func renderDashboard(c echo.Context) error {
	boards, err := db.GetBoards()
	if err != nil { return err }
	accounts, err := db.GetAccounts()
	if err != nil { return err }
	themes, _ := db.GetThemes()
	data := struct {
		Accounts	[]db.Account
		Boards		[]db.Board
		Config		config.Config
		Theme		string
		Themes		[]string
		Bans		[]db.Ban
		UserThemes	[]db.Theme
		Header		any
	}{
		Accounts: accounts,
		Boards: boards,
		Config: config.Cfg,
		Bans:	db.BanList,
		Themes: getThemes(),
		UserThemes: themes,
		Header: header(c),
	}
	return render("dashboard.html", data, c)
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
