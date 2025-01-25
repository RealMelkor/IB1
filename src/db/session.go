package db

import (
	"gorm.io/gorm"
	"time"
	"log"
	"IB1/util"
)

type KeyValue struct {
	Value		any	`gorm:"serializer:json"`
	Key		string
	Creation	time.Time
	Token		string
}

type Session struct {
	AccountID	uint
	Account		Account
	Token		string `gorm:"unique"`
}

// cached session tokens
var sessions = util.SafeMap[Account]{}

func createSession(account Account) (string, error) {
	token, err := util.NewToken()
	if err != nil { return "", err }
	err = db.Create(&Session{
		Account: account,
		AccountID: account.ID,
		Token: token,
	}).Error
	return token, err
}

type KeyValues map[string]KeyValue

func SaveSessions(sessions *util.SafeMap[KeyValues]) error {
	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&KeyValue{})
	sessions.Iter(func(key string, values KeyValues)(KeyValues, bool) {
		for _, v := range values {
			v.Token = key
			if err := db.Create(v).Error; err != nil {
				log.Println(err)
			}
		}
		return values, true
	})
	return nil
}

func LoadSessions(sessions *util.SafeMap[KeyValues]) error {
	rows, err := db.Model(&KeyValue{}).Rows()
	if err != nil { return err }
	defer rows.Close()
	for rows.Next() {
		var v KeyValue
		if err := db.ScanRows(rows, &v); err != nil { return err }
		m, ok := sessions.Get(v.Token)
		if !ok {
			m = KeyValues{}
		}
		m[v.Key] = v
		sessions.Set(v.Token, m)
	}
	return nil
}
