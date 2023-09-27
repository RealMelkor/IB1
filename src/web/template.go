package web

import (
	_ "embed"
	"html/template"
	"bytes"
	"strings"

	"IB1/db"
)

//go:embed html/header.gohtml
var headerRaw string

//go:embed html/footer.gohtml
var footerRaw string

//go:embed html/catalog.gohtml
var catalogRaw string

//go:embed html/thread.gohtml
var threadRaw string

//go:embed static/favicon.png
var favicon string

//go:embed static/style.css
var stylesheet string

var header string
var footer string
var catalogTemplate *template.Template
var threadTemplate *template.Template

func initTemplate() error {

	tmpl, err := template.New("catalog").Parse(catalogRaw)
	if err != nil { return err }
	catalogTemplate = tmpl

	tmpl, err = template.New("thread").Parse(threadRaw)
	if err != nil { return err }
	threadTemplate = tmpl

	return refreshTemplate()
}

func refreshTemplate() error {

	var buf bytes.Buffer

	tmpl, err := template.New("header").Parse(headerRaw)
	if err != nil { return err }

	boards, err := db.GetBoards()
	if err != nil { return err }

	data := struct {
		Title	string
		Lang	string
		Boards	[]db.Board
        }{
                "IB1",
                "en",
		boards,
        }
	err = tmpl.Execute(&buf, data)
	if err != nil { return err }

	header = buf.String()
	buf.Reset()

	tmpl, err = template.New("footer").Parse(footerRaw)
	if err != nil { return err }

	err = tmpl.Execute(&buf, nil)
	if err != nil { return err }

	footer = buf.String()

	return nil
}

func renderCatalog(board db.Board) (string, error) {

	var buf bytes.Buffer

	err := catalogTemplate.Execute(&buf, board)
	if err != nil { return "", err }
	
	return buf.String(), nil
}

func renderThread(thread db.Thread) (string, error) {

	var buf bytes.Buffer

	err := threadTemplate.Execute(&buf, thread)
	if err != nil { return "", err }
	
	return buf.String(), nil
}

func parseContent(content string) template.HTML {
	content = template.HTMLEscapeString(content)
	content = strings.Replace(content, "\n", "<br>", -1)
	return template.HTML(content)
}
