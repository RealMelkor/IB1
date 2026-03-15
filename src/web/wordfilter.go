package web

import (
	"regexp"

	"IB1/db"
	"IB1/filter"
)

func createWordfilter(from, to string, enabled bool) error {
	_, err := regexp.Compile(from)
	if err != nil {
		return err
	}
	filter.Wordfilters.Refresh()
	return db.Wordfilter{}.Add(db.Wordfilter{
		From:     from,
		To:       to,
		Disabled: !enabled,
	})
}

func deleteWordfilter(id int) error {
	filter.Wordfilters.Refresh()
	return db.Wordfilter{}.RemoveID(id, db.Wordfilter{})
}

func updateWordfilter(id int, from, to string, enabled bool) error {
	_, err := regexp.Compile(from)
	if err != nil {
		return err
	}
	filter.Wordfilters.Refresh()
	return db.Wordfilter{}.Update(id, db.Wordfilter{
		From:     from,
		To:       to,
		Disabled: !enabled,
	})
}
