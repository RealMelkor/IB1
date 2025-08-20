package db

import (
	"errors"
	"net"
	"time"

	"github.com/yl2chen/cidranger"
	"gorm.io/gorm"
)

type Ban struct {
	gorm.Model
	CIDR    string
	Expiry  int64
	BoardID *uint
	Board   Board
}

var ranger = map[uint]cidranger.Ranger{}

func GetBanList() ([]Ban, error) {
	var list = []Ban{}
	tx := db.Find(&list)
	return list, tx.Error
}

func LoadBanList() error {
	var BanList = []Ban{}
	tx := db.Find(&BanList)
	if tx.Error != nil {
		return tx.Error
	}
	for _, v := range BanList {
		id := uint(0)
		if v.BoardID != nil {
			id = *v.BoardID
		}
		_, ok := ranger[id]
		if !ok {
			ranger[id] = cidranger.NewPCTrieRanger()
		}
		_, cidr, _ := net.ParseCIDR(v.CIDR)
		ranger[id].Insert(cidranger.NewBasicRangerEntry(*cidr))
	}
	return nil
}

func IsBanned(_ip string, boardID uint) error {
	ip := net.ParseIP(_ip)
	if ip == nil {
		return errors.New("invalid ip")
	}
	if len(ranger) < 1 {
		return nil
	}
	v, err := ranger[0].Contains(ip)
	if err != nil {
		return err
	}
	if v {
		return errors.New("banned")
	}
	v, err = ranger[boardID].Contains(ip)
	if err != nil {
		return err
	}
	if v {
		return errors.New("banned")
	}
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

func BanIP(ip string, duration int64, boardID uint) error {
	_, _, err := net.ParseCIDR(ip)
	if err != nil {
		_ip := net.ParseIP(ip)
		if _ip == nil {
			return errors.New("invalid ip")
		}
		ip = _ip.String() + "/32"
	}
	v := &boardID
	if boardID == 0 {
		v = nil
	}
	ban := Ban{
		CIDR:    ip,
		Expiry:  time.Now().Unix() + duration,
		BoardID: v,
	}
	if err := db.Create(&ban).Error; err != nil {
		return err
	}
	_, cidr, err := net.ParseCIDR(ip)
	if err != nil {
		return err
	}
	_, ok := ranger[boardID]
	if !ok {
		ranger[boardID] = cidranger.NewPCTrieRanger()
	}
	return ranger[boardID].Insert(cidranger.NewBasicRangerEntry(*cidr))
}

func RemoveBan(id uint) error {
	err := db.Unscoped().Delete(&Ban{}, id).Error
	if err != nil {
		return err
	}
	var ban Ban
	if err := db.Find(&ban, id).Error; err != nil {
		return err
	}
	_, cidr, err := net.ParseCIDR(ban.CIDR)
	if err != nil {
		return err
	}
	ranger[ban.ID].Remove(*cidr)
	return nil
}
