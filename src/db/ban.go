package db

import (
	"time"
	"errors"
)

func LoadBanList() error {
	var bans []Ban
	tx := db.Find(&bans)
	if tx.Error != nil {  return tx.Error }
	for _, v := range bans {
		BanList[v.IP] = v
	}
	return nil
}

func IsBanned(ip string) error {
	v, ok := BanList[ip]
	if !ok { return nil }
	if v.Expiry < time.Now().Unix() {
		delete(BanList, ip)
		return nil
	}
	return errors.New("banned")
}

func BanIP(ip string) error {
	ban := Ban{IP: ip, Expiry: time.Now().Unix() + 3600}
	BanList[ip] = ban
	return db.Create(ban).Error
}
