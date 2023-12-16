package web

import (
	_ "embed"
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

//go:embed html/header.gohtml
var headerRaw string

//go:embed html/footer.gohtml
var footerRaw string

//go:embed html/index.gohtml
var indexRaw string

//go:embed html/board.gohtml
var boardRaw string

//go:embed html/catalog.gohtml
var catalogRaw string

//go:embed html/threadtemplate.gohtml
var threadTemplateRaw string

//go:embed html/thread.gohtml
var threadRaw string

//go:embed html/top.gohtml
var topRaw string

//go:embed html/bottom.gohtml
var bottomRaw string

//go:embed html/newthread.gohtml
var newThreadRaw string

//go:embed static/favicon.png
var favicon string

//go:embed static/style.css
var stylesheet string

var header string
var footer string
var boardTemplate *template.Template
var catalogTemplate *template.Template
var threadTemplate *template.Template
var indexTemplate *template.Template

func initTemplate() error {

	blocks := topRaw + bottomRaw + newThreadRaw + threadTemplateRaw

	tmpl, err := template.New("board").Parse(blocks + boardRaw)
	if err != nil { return err }
	boardTemplate = tmpl

	tmpl, err = template.New("catalog").Parse(blocks + catalogRaw)
	if err != nil { return err }
	catalogTemplate = tmpl

	tmpl, err = template.New("thread").Parse(blocks + threadRaw)
	if err != nil { return err }
	threadTemplate = tmpl

	tmpl, err = template.New("index").Parse(indexRaw)
	if err != nil { return err }
	indexTemplate = tmpl

	return refreshTemplate()
}

func refreshTemplate() error {

	var buf bytes.Buffer

	m := minify.New()
	m.AddFunc("text/html", mhtml.Minify)

	tmpl, err := template.New("header").Parse(headerRaw)
	if err != nil { return err }

	boards, err := db.GetBoards()
	if err != nil { return err }

	data := struct {
		Title	string
		Lang	string
		Boards	[]db.Board
        }{
                config.Cfg.Home.Title,
                config.Cfg.Home.Language,
		boards,
        }
	err = tmpl.Execute(&buf, data)
	if err != nil { return err }

	header = buf.String()

	res, err := m.String("text/html", header)
	if err != nil { return err }
	header = res

	buf.Reset()

	tmpl, err = template.New("footer").Parse(footerRaw)
	if err != nil { return err }

	err = tmpl.Execute(&buf, nil)
	if err != nil { return err }

	footer = buf.String()

	res, err = m.String("text/html", footer)
	if err != nil { return err }
	footer = res

	return nil
}

func renderIndex() (string, error) {

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
	err := indexTemplate.Execute(&buf, data)
	if err != nil { return "", err }
	
	return buf.String(), nil
}

func renderBoard(board db.Board) (string, error) {

	var buf bytes.Buffer

	data := struct {
		Board	db.Board
		Captcha	bool
	}{
		Board: board,
		Captcha: config.Cfg.Captcha.Enabled,
	}

	err := boardTemplate.Execute(&buf, data)
	if err != nil { return "", err }
	
	return buf.String(), nil
}

func renderCatalog(board db.Board) (string, error) {

	var buf bytes.Buffer

	data := struct {
		Board	db.Board
		Captcha	bool
	}{
		Board: board,
		Captcha: config.Cfg.Captcha.Enabled,
	}

	err := catalogTemplate.Execute(&buf, data)
	if err != nil { return "", err }
	
	return buf.String(), nil
}

func renderThread(thread db.Thread) (string, error) {

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

	err := threadTemplate.Execute(&buf, data)
	if err != nil { return "", err }
	
	return buf.String(), nil
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
		res, err := m.String("text/css", stylesheet)
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
		res, err := m.String("text/html", tmp)
		if err != nil { return nil, err }
		indexCache = []byte(header + res + footer)
	}
	return indexCache, nil
}
