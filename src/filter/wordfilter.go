package filter

import (
	"regexp"

	"IB1/db"
	"IB1/util"
)

var Wordfilters = util.SafeObj[[]db.Wordfilter]{
	Reload: getWordfilters,
}

func getWordfilters() ([]db.Wordfilter, error) {
	v, err := db.Wordfilter{}.GetAll()
	if err != nil {
		return nil, err
	}
	filters := []db.Wordfilter{}
	for _, v := range v {
		if !v.Disabled {
			v.Regexp = regexp.MustCompile(v.From)
			filters = append(filters, v)
		}
	}
	return filters, nil
}

func FilterText(in string) (string, error) {
	filters, err := Wordfilters.Get()
	if err != nil {
		return "", err
	}
	for _, filter := range filters {
		in = filter.Regexp.ReplaceAllString(in, filter.To)
	}
	return in, nil
}
