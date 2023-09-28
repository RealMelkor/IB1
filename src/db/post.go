package db

import (
	"strings"
	"fmt"
	"strconv"
	"time"
)

func (post Post) FormatTimestamp() string {
	tm := time.Unix(post.Timestamp, 0).UTC()
	return fmt.Sprintf("%02d/%02d/%d (%s) %02d:%02d:%02d UTC",
		tm.Month(), tm.Day(), tm.Year() % 1000,
		tm.Weekday().String()[0:3],
		tm.Hour(), tm.Minute(), tm.Second())
}

func (post Post) FormatAge() string {
	const minute = 60
	const hour = minute * 60
	const day = hour * 24
	const month = day * 30
	const year = month * 12
	seconds := time.Now().Unix() - post.Timestamp
	var i int64
	var str string
	if seconds > year * 2 {
		i = seconds / year
		str = "year"
	} else if seconds > month {
		i = seconds / month
		str = "month"
	} else if seconds > day {
		i = seconds / day
		str = "day"
	} else if seconds > hour {
		i = seconds / hour
		str = "hour"
	} else if seconds > minute {
		i = seconds / minute
		str = "minute"
	} else {
		i = seconds
		str = "second"
	}
	str = strconv.Itoa(int(i)) + " " + str
	if i > 1 { str += "s" }
	str += " ago"
	return str
}

func (post Post) Thumbnail() string {
	if post.Media == "" { return "" }
	i := strings.LastIndex(post.Media, ".")
	if i < 1 { return "" }
	return post.Media[0:i] + ".png"
}
