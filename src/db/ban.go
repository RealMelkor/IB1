package db

import (
	"time"
	"net"
	"errors"

	"github.com/yl2chen/cidranger"
	"gorm.io/gorm"
)

type Ban struct {
	gorm.Model
	CIDR		string
	Expiry		int64
}

var BanList = []Ban{}
var ranger cidranger.Ranger

func LoadBanList() error {
	tx := db.Find(&BanList)
	if tx.Error != nil {  return tx.Error }
	ranger = cidranger.NewPCTrieRanger()
	for _, v := range BanList {
		_, cidr, _ := net.ParseCIDR(v.CIDR)
		ranger.Insert(cidranger.NewBasicRangerEntry(*cidr))
	}
	return nil
}

func IsBanned(_ip string) error {
	ip := net.ParseIP(_ip)
	if ip == nil { return errors.New("invalid ip") }
	v, err := ranger.Contains(ip)
	if err != nil { return err }
	if v { return errors.New("banned") }
	return nil
}

func (ban Ban) From() string {
	return ban.CreatedAt.UTC().Format(time.RFC1123)
}

func (ban Ban) To() string {
	return time.Unix(ban.Expiry, 0).UTC().Format(time.RFC1123)
}

func (ban Ban) String() string {
	return ban.CIDR
}

func BanIP(ip string, duration int64) error {
	_, _, err := net.ParseCIDR(ip)
	if err != nil {
		_ip := net.ParseIP(ip)
		if _ip == nil { return errors.New("invalid ip") }
		ip = _ip.String() + "/32"
	}
	ban := Ban{
		CIDR: ip,
		Expiry: time.Now().Unix() + duration,
	}
	if err := db.Create(&ban).Error; err != nil { return err }
	BanList = append(BanList, ban)
	_, cidr, err := net.ParseCIDR(ip)
	if err != nil { return err }
	return ranger.Insert(cidranger.NewBasicRangerEntry(*cidr))
}

func RemoveBan(id uint) error {
	err := db.Unscoped().Delete(&Ban{}, id).Error
	if err != nil { return err }
	for i, v := range BanList {
		if v.ID == id {
			_, cidr, err := net.ParseCIDR(v.CIDR)
			if err != nil { return err }
			ranger.Remove(*cidr)
			BanList[i] = BanList[len(BanList) - 1]
			BanList = BanList[:len(BanList) - 1]
			return nil
		}
	}
	return nil
}
