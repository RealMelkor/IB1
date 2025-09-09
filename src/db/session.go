package db

import (
	"IB1/util"
	"gorm.io/gorm"
	"log"
	"time"
)

type KeyValue struct {
	Value    any `gorm:"serializer:json"`
	Key      string
	Creation time.Time
	Token    string
}

type Session struct {
	AccountID uint
	Account   Account
	Token     string `gorm:"unique"`
}

type SessionKey struct {
	session	string
	key	string
}

func GetSessionKey(session, key string) SessionKey {
	return SessionKey{session: session, key: key}
}

// cached session tokens
var sessions = util.SafeMap[string, Account]{}

func createSession(account Account) (string, error) {
	token, err := util.NewToken()
	if err != nil {
		return "", err
	}
	err = db.Create(&Session{
		Account:   account,
		AccountID: account.ID,
		Token:     token,
	}).Error
	return token, err
}

func SaveSessions(sessions *util.SafeMap[SessionKey, KeyValue]) error {
	db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&KeyValue{})
	sessions.Iter(func(key SessionKey, v KeyValue) (KeyValue, bool) {
		v.Token = key.key
		if err := db.Create(v).Error; err != nil {
			log.Println(err)
		}
		return v, true
	})
	return nil
}

func LoadSessions(sessions *util.SafeMap[SessionKey, KeyValue]) error {
	rows, err := db.Model(&KeyValue{}).Rows()
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var v KeyValue
		if err := db.ScanRows(rows, &v); err != nil {
			return err
		}
		sessions.Set(GetSessionKey(v.Token, v.Key), v)
	}
	return nil
}
