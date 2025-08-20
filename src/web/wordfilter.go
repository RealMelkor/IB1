package web

import (
	"strconv"
	"regexp"

	"IB1/db"
	"IB1/util"

	"github.com/labstack/echo/v4"
)

func createWordfilter(c echo.Context) error {
	from, hasFrom := getPostForm(c, "from")
        if !hasFrom { return errInvalidForm }
	_, err := regexp.Compile(from)
	if err != nil { return err }
	to, hasTo := getPostForm(c, "to")
        if !hasTo { return errInvalidForm }
	enabled, _ := getPostForm(c, "enabled")
	disabled := enabled != "on"
	wordfilters.Refresh()
	return db.Wordfilter{}.Add(db.Wordfilter{
		From: from,
		To: to,
		Disabled: disabled,
	})
}

func deleteWordfilter(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil { return errInvalidID }
	wordfilters.Refresh()
	return db.Wordfilter{}.RemoveID(id, db.Wordfilter{})
}

func updateWordfilter(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil { return errInvalidID }
	from, hasFrom := getPostForm(c, "from")
        if !hasFrom { return errInvalidForm }
	_, err = regexp.Compile(from)
	if err != nil { return err }
	to, _ := getPostForm(c, "to")
	enabled, _ := getPostForm(c, "enabled")
	disabled := enabled != "on"
	wordfilters.Refresh()
	return db.Wordfilter{}.Update(id, db.Wordfilter{
		From: from,
		To: to,
		Disabled: disabled,
	})
}

var wordfilters = util.SafeObj[[]db.Wordfilter]{
	Reload: getWordfilters,
}

func getWordfilters() ([]db.Wordfilter, error) {
	v, err := db.Wordfilter{}.GetAll()
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
	for _, filter := range filters {
		in = filter.Regexp.ReplaceAllString(in, filter.To)
	}
	return in, nil
}
