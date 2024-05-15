package web

import (
	"embed"
	"html/template"
	"strings"
	"strconv"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/gin-gonic/gin"

	"IB1/db"
	"IB1/config"
)

//go:embed html/*.gohtml
var templatesFS embed.FS

//go:embed static/*
var static embed.FS

//go:embed static/favicon.png
var favicon []byte

var templates *template.Template

func initTemplate() error {
	var err error
	funcs := template.FuncMap{
		"thread": func(thread db.Thread, account db.Account) any {
			data := struct {
				Account db.Account
				Thread db.Thread
			}{
				Account: account, Thread: thread,
			}
			return data
		},
	}
	templates, err = template.New("gmi").Funcs(funcs).
				ParseFS(templatesFS, "html/*.gohtml")
	if err != nil { return err }
	if err := minifyStylesheet(); err != nil { return err }
	return nil
}

func header(c *gin.Context) any {
	var boards []db.Board
	for _, v := range db.Boards {
		boards = append([]db.Board{v}, boards...)
	}
	var account db.Account
	account.Logged = false
	logged := false
	token, _ := c.Cookie("session_token")
	if token != "" {
		var err error
		account, err = db.GetAccountFromToken(token)
		if err == nil { logged = true }
	}
	data := struct {
		Title	string
		Lang	string
		Url	string
		Theme	string
		Themes	[]string
		Logged	bool
		Account	db.Account
		Boards	[]db.Board
	}{
		config.Cfg.Home.Title,
		config.Cfg.Home.Language,
		c.Request.RequestURI,
		getTheme(c),
		getThemes(),
		logged,
		account,
		boards,
	}
	return data
}

func renderIndex(c *gin.Context) error {
	data := struct {
		Boards		map[string]db.Board
		Title		string
		Description	string
		Header		any
	}{
		Boards: db.Boards,
		Title: config.Cfg.Home.Title,
		Description: config.Cfg.Home.Description,
		Header: header(c),
	}
	return render("index.gohtml", data, c)
}

func renderDashboard(c *gin.Context) error {
	boards, err := db.GetBoards()
	if err != nil { return err }
	data := struct {
		Boards		[]db.Board
		Config		config.Config
		Theme		string
		Themes		[]string
		Header		any
	}{
		Boards: boards,
		Config: config.Cfg,
		Themes: getThemes(),
		Header: header(c),
	}
	return render("dashboard.gohtml", data, c)
}

func renderBoard(board db.Board, threads int, c *gin.Context) error {
	pages := []int{}
	count := (threads + 3) / 4
	for i := 0; i < count; i++ { pages = append(pages, i) }
	data := struct {
		Board	db.Board
		Captcha	bool
		Pages	[]int
		Header	any
	}{
		Board: board,
		Captcha: config.Cfg.Captcha.Enabled,
		Pages: pages,
		Header: header(c),
	}
	return render("board.gohtml", data, c)
}

func renderCatalog(board db.Board, c *gin.Context) error {
	data := struct {
		Board	db.Board
		Captcha	bool
		Header	any
	}{
		Board: board,
		Captcha: config.Cfg.Captcha.Enabled,
		Header: header(c),
	}
	return render("catalog.gohtml", data, c)
}

func renderThread(thread db.Thread, c *gin.Context) error {
	data := struct {
		Board	db.Board
		Thread	db.Thread
		Captcha	bool
		Header	any
	}{
		Board: thread.Board,
		Thread: thread,
		Captcha: config.Cfg.Captcha.Enabled,
		Header: header(c),
	}
	return render("thread.gohtml", data, c)
}

func renderLogin(c *gin.Context, err string) error {
	data := struct {
		LoginError	string
		Captcha		bool
		Header	any
	}{
		LoginError: err,
		Captcha: config.Cfg.Captcha.Enabled,
		Header: header(c),
	}
	return render("login.gohtml", data, c)
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
		str := "<a href=\"#" + number + "\">&gt;&gt;" + number + "</a>"
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

func parseContent(content string, thread uint) (template.HTML, []int) {
	content = template.HTMLEscapeString(content)
	content = strings.Replace(content, "\n", "<br>", -1)
	content, refs := addLinks(content, thread)
	content = addGreentext(content)
	return template.HTML(content), refs
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
