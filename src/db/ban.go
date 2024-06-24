package db

import (
	"gorm.io/gorm"
	"time"
	"net"
	"strconv"
	"encoding/binary"
	"bytes"
	"errors"
)

type Ban struct {
	gorm.Model
	IP		uint32
	Mask		uint32
	Expiry		int64
}

var BanList = []Ban{}

func LoadBanList() error {
	tx := db.Find(&BanList)
	if tx.Error != nil {  return tx.Error }
	return nil
}

func IsBanned(_ip string) error {
	ip, err := parseIP(_ip)
	if err != nil { return err }
	for _, v := range BanList {
		if ip & v.Mask == v.IP & v.Mask {
			return errors.New("banned")
		}
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
	ip := int(ban.IP)
	b0 := strconv.Itoa((ip>>24)&0xff)
	b1 := strconv.Itoa((ip>>16)&0xff)
	b2 := strconv.Itoa((ip>>8)&0xff)
	b3 := strconv.Itoa((ip& 0xff))
	mask := 0
	for i := ban.Mask; i > 0; i = i<<1 {
		mask += 1
	}
	return b0 + "." + b1 + "." + b2 + "." + b3 + "/" + strconv.Itoa(mask)
}

func parseIP(ip string) (uint32, error) {
	_ip := net.ParseIP(ip)
	if _ip == nil || _ip.To4() == nil {
		return 0, errors.New("invalid ip")
	}
	var i uint32
	binary.Read(bytes.NewBuffer(_ip.To4()), binary.BigEndian, &i)
	return i, nil
}

func parseCIDR(ip string) (uint32, uint32, error) {
	_ip, _net, err := net.ParseCIDR(ip)
	if err != nil {
		i, err := parseIP(ip)
		return i, 0xFFFFFFFF, err
	}
	n, _ := _net.Mask.Size()
	mask := 0
	for n > 0 {
		n--
		mask |= 0x80000000 >> n
	}
	if _ip == nil || _ip.To4() == nil {
		return 0, 0, errors.New("invalid ip")
	}
	var i uint32
	binary.Read(bytes.NewBuffer(
		_ip.To4()), binary.BigEndian, &i)
	return i, uint32(mask), nil
}

func BanIP(ip string, duration int64) error {
	_ip, mask, err := parseCIDR(ip)
	if err != nil { return err }
	ban := Ban{IP: _ip, Mask: mask, Expiry: time.Now().Unix() + duration}
	if err := db.Create(&ban).Error; err != nil { return err }
	BanList = append(BanList, ban)
	return nil
}

func remove(s []int, i int) []int {
	s[i] = s[len(s) - 1]
	return s[:len(s) - 1]
}

func RemoveBan(id uint) error {
	err := db.Unscoped().Delete(&Ban{}, id).Error
	if err != nil { return err }
	for i, v := range BanList {
		if v.ID == id {
			BanList[i] = BanList[len(BanList) - 1]
			BanList = BanList[:len(BanList) - 1]
			return nil
		}
	}
	return nil
}
