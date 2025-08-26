package web

import (
	"io"
	"strings"
	"strconv"
	"net/url"
	"html/template"
	"log"

	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"

	"IB1/db"
	"IB1/config"
)

func removeDuplicate[T comparable](sliceList []T) []T {
	allKeys := make(map[T]bool)
	list := []T{}
	for _, item := range sliceList {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}

func parseLinks(content string) string {
	links := []string{}
	for v := range strings.FieldsSeq(content) {
		if !strings.HasPrefix(v, "https://") {
			continue
		}
		links = append(links, v)
	}
	links = removeDuplicate(links)
	for _, v := range links {
		res, err := url.Parse(v)
		if err != nil {
			log.Println(v, err)
			continue
		}
		s := "<a href=\"" + res.String() + "\">" + res.String() + "</a>"
		content = strings.ReplaceAll(content, v, s)
	}
	return content
}

func parseRefs(content string, thread uint) (string, []int) {
	const quote = "&gt;&gt;"
	var refs []int
	for i := strings.Index(content, quote); i >= 0 &&
		i+len(quote) < len(content); {

		index := strings.Index(content[i:], quote)
		if index < 0 {
			break
		}
		i += index + len(quote)

		j := i
		length := len(content)
		for ; j < length; j++ {
			if content[j] < '0' || content[j] > '9' {
				break
			}
		}
		if j < length && content[j] != ' ' && content[j] != '\n' &&
			content[j] == '\t' {
			continue
		}
		number := content[i:j]
		n, err := strconv.Atoi(number)
		if err != nil {
			continue
		}
		if _, err := db.GetPost(thread, n); err != nil {
			continue
		}
		refs = append(refs, n)
		str := "<a class=\"l-" + number + "\" href=\"#" + number +
			"\">&gt;&gt;" + number + "</a>"
		content = content[:i-len(quote)] + str + content[j:]
		i += len(str) - len(quote)
	}

	return content, removeDuplicate(refs)
}

func addGreentext(content string) string {
	const br = "<br>"
	content = strings.ReplaceAll(content, "\r", "")
	length := len(content)
	next := 0
	for i := 0; i >= 0 && i < length; i = next {
		next = strings.Index(content[i:], br)
		if next == -1 {
			next = length
		} else {
			next += i + len(br)
		}
		if strings.Index(content[i:next], "&gt;&gt;") == 0 {
			continue
		}
		if strings.Index(content[i:next], "&gt;") != 0 {
			continue
		}
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
	if i == 0 {
		return ""
	}
	return string(res[:i])
}

func parseContent(content string, thread uint) (template.HTML, []int) {
	if config.Cfg.Post.AsciiOnly {
		content = asciiOnly(content)
	}
	content = template.HTMLEscapeString(content)
	content = strings.Replace(content, "\n", "<br>", -1)
	content = parseLinks(content)
	content, refs := parseRefs(content, thread)
	content = addGreentext(content)
	return template.HTML(content), refs
}

func minifyCSS(in []byte) ([]byte, error) {
	m := minify.New()
	m.AddFunc("text/css", css.Minify)
	res, err := m.Bytes("text/css", in)
	if err != nil {
		return nil, err
	}
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
	if err != nil {
		return err
	}
	res, err := m.Bytes("text/css", data)
	if err != nil {
		return err
	}
	stylesheet = res
	return nil
}
