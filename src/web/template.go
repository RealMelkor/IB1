package web

import (
	"embed"
	"html/template"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	mhtml "github.com/tdewolff/minify/v2/html"
	"bytes"
	"strings"
	"strconv"

	"IB1/db"
	"IB1/config"
)

//go:embed html/*.gohtml
var templatesFS embed.FS

//go:embed static/*
var static embed.FS

//go:embed static/favicon.png
var favicon []byte

var footer []byte
var header []byte

var templates *template.Template

func initTemplate() error {
	var err error
	templates, err = template.New("gmi").
				ParseFS(templatesFS, "html/*.gohtml")
	if err != nil { return err }
	return refreshTemplate()
}

func refreshTemplate() error {

	var buf bytes.Buffer

	var boards []db.Board
	for _, v := range db.Boards {
		boards = append([]db.Board{v}, boards...)
	}

	data := struct {
		Title	string
		Lang	string
		Boards	[]db.Board
        }{
                config.Cfg.Home.Title,
                config.Cfg.Home.Language,
		boards,
        }

	err := templates.Lookup("header.gohtml").Execute(&buf, data)
	if err != nil { return err }

	m := minify.New()
	m.AddFunc("text/html", mhtml.Minify)
	res, err := m.Bytes("text/html", buf.Bytes())
	if err != nil { return err }
	header = res

	buf.Reset()

	err = templates.Lookup("footer.gohtml").Execute(&buf, data)
	if err != nil { return err }

	res, err = m.Bytes("text/html", buf.Bytes())
	if err != nil { return err }
	footer = res

	return nil
}

func renderIndex() ([]byte, error) {

	var buf bytes.Buffer

	data := struct {
		Boards		map[string]db.Board
		Title		string
		Description	string
	}{
		Boards: db.Boards,
		Title: config.Cfg.Home.Title,
		Description: config.Cfg.Home.Description,
	}
	err := templates.Lookup("index.gohtml").Execute(&buf, data)
	if err != nil { return nil, err }

	return buf.Bytes(), nil
}

func renderBoard(board db.Board) ([]byte, error) {

	var buf bytes.Buffer

	data := struct {
		Board	db.Board
		Captcha	bool
	}{
		Board: board,
		Captcha: config.Cfg.Captcha.Enabled,
	}

	err := templates.Lookup("board.gohtml").Execute(&buf, data)
	if err != nil { return nil, err }
	
	return buf.Bytes(), nil
}

func renderCatalog(board db.Board) ([]byte, error) {

	var buf bytes.Buffer

	data := struct {
		Board	db.Board
		Captcha	bool
	}{
		Board: board,
		Captcha: config.Cfg.Captcha.Enabled,
	}

	err := templates.Lookup("catalog.gohtml").Execute(&buf, data)
	if err != nil { return nil, err }
	
	return buf.Bytes(), nil
}

func renderThread(thread db.Thread) ([]byte, error) {

	var buf bytes.Buffer

	data := struct {
		Board	db.Board
		Thread	db.Thread
		Captcha	bool
	}{
		Board: thread.Board,
		Thread: thread,
		Captcha: config.Cfg.Captcha.Enabled,
	}

	err := templates.Lookup("thread.gohtml").Execute(&buf, data)
	if err != nil { return nil, err }
	
	return buf.Bytes(), nil
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
		if (j < length && content[j] != ' ' && content[j] != '\n' &&
				content[j] == '\t') {
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

var stylesheetCached []byte = nil
func minifyStylesheet() ([]byte, error) {
	if stylesheetCached == nil {
		m := minify.New()
		m.AddFunc("text/css", css.Minify)
		data, err := static.ReadFile("static/style.css")
		if err != nil { return nil, err }
		res, err := m.String("text/css", string(data))
		if err != nil { return nil, err }
		stylesheetCached = []byte(res)
	}
	return stylesheetCached, nil
}

var indexCache[]byte = nil
func minifyIndex() ([]byte, error) {
	if indexCache == nil {
		tmp, err := renderIndex()
		if err != nil { return nil, err }
		m := minify.New()
		m.AddFunc("text/html", mhtml.Minify)
		res, err := m.Bytes("text/html", tmp)
		if err != nil { return nil, err }
		indexCache = append(header, append(res, footer...)...)
	}
	return indexCache, nil
}
