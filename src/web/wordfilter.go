package web

import (
	"strconv"
	"strings"
	"regexp"

	"IB1/db"
	"IB1/util"

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
	wordfilters.Refresh()
	return db.AddWordfilter(from, to, disabled)
}

func deleteWordfilter(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil { return invalidID }
	wordfilters.Refresh()
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
	wordfilters.Refresh()
	return db.UpdateWordfilter(id, from, to, disabled)
}

var wordfilters = util.SafeObj[[]db.Wordfilter]{
	Reload: getWordfilters,
}

func getWordfilters() ([]db.Wordfilter, error) {
	v, err := db.GetWordfilters()
	if err != nil { return nil, err }
	filters := []db.Wordfilter{}
	for _, v := range v {
		if !v.Disabled {
			v.Regexp = regexp.MustCompile(v.From)
			filters = append(filters, v)
		}
	}
	return filters, nil
}

func filterText(in string) (string, error) {
	filters, err := wordfilters.Get()
	if err != nil { return "", err }
	if len(filters) == 0 { return in, nil }
	words := strings.Split(in, " ")
	res := ""
	for _, v := range words {
		for _, filter := range filters {
			v = filter.Regexp.ReplaceAllString(v, filter.To)
		}
		res += v + " "
	}
	return res, nil
}
