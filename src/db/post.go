package db

import (
	"gorm.io/gorm"
	"html/template"
	"strings"
	"fmt"
	"strconv"
	"time"
	"sync"
)

func (post Post) FormatTimestamp() string {
	tm := time.Unix(post.Timestamp, 0).UTC()
	return fmt.Sprintf("%02d/%02d/%d (%s) %02d:%02d:%02d UTC",
		tm.Month(), tm.Day(), tm.Year(),
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

func (post Post) ReferredBy() string {
	return ""
}

var newPostLock sync.Mutex
func CreatePost(thread Thread, content template.HTML, name string,
		media string, custom *gorm.DB) (int, error) {
	if custom == nil { custom = db }
	if name == "" { name = DefaultName }
	if dbType == TYPE_SQLITE {
		newPostLock.Lock()
	}
	number := -1
	err := custom.Transaction(func(tx *gorm.DB) error {

		tx.Select("Posts").Find(&thread.Board)

		err := tx.Model(&thread.Board).
			Update("Posts", thread.Board.Posts + 1).Error
		if err != nil { return err }

		ret := tx.Create(&Post{
			Board: thread.Board, Thread: thread, Name: name,
			Content: content, Timestamp: time.Now().Unix(),
			Number: thread.Board.Posts, Media: media,
		})
		if ret.Error != nil { return err }

		number = thread.Board.Posts

		return nil
	})
	if dbType == TYPE_SQLITE {
		newPostLock.Unlock()
	}
	return number, err
}

func GetPost(thread Thread, number int) (Post, error) {
	var post Post
	err := db.First(
		&post, "thread_id = ?, number = ?", thread.ID, number).Error;
	if err != nil {
		return Post{}, err
	}
	return post, nil
}
