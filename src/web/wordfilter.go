package web

import (
	"strconv"
	"strings"
	"regexp"

	"IB1/db"

	"github.com/labstack/echo/v4"
)

func createWordfilter(c echo.Context) error {
	from, hasFrom := getPostForm(c, "from")
        if !hasFrom { return invalidForm }
	_, err := regexp.Compile(from)
	if err != nil { return err }
	to, hasTo := getPostForm(c, "to")
        if !hasTo { return invalidForm }
	enabled, _ := getPostForm(c, "enabled")
	disabled := enabled != "on"
	wordfilterUpdate = true
	return db.AddWordfilter(from, to, disabled)
}

func deleteWordfilter(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil { return invalidID }
	wordfilterUpdate = true
	return db.RemoveWordfilter(id)
}

func updateWordfilter(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil { return invalidID }
	from, hasFrom := getPostForm(c, "from")
        if !hasFrom { return invalidForm }
	_, err = regexp.Compile(from)
	if err != nil { return err }
	to, _ := getPostForm(c, "to")
	enabled, _ := getPostForm(c, "enabled")
	disabled := enabled != "on"
	wordfilterUpdate = true
	return db.UpdateWordfilter(id, from, to, disabled)
}

var wordfilters = []db.Wordfilter{}
var wordfilterUpdate = true
func getWordfilters() ([]db.Wordfilter, error) {
	if wordfilterUpdate {
		v, err := db.GetWordfilters()
		if err != nil { return nil, err }
		filters := []db.Wordfilter{}
		for _, v := range v {
			if !v.Disabled {
				v.Regexp = regexp.MustCompile(v.From)
				filters = append(filters, v)
			}
		}
		wordfilters = filters
		wordfilterUpdate = false

	}
	return wordfilters, nil
}

func filterText(in string) (string, error) {
	filters, err := getWordfilters()
	if err != nil { return "", err }
	if len(filters) == 0 { return in, nil }
	words := strings.Split(in, " ")
	res := ""
	for _, v := range words {
		for _, filter := range filters {
			v = filter.Regexp.ReplaceAllString(v, filter.To)
			//v = strings.Replace(v, filter.From, filter.To, -1)
		}
		res += v + " "
	}
	return res, nil
}
