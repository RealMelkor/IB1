package dnsbl

import (
	"errors"
	"net"
	"strings"
	"time"

	"IB1/db"
	"IB1/util"
)

const retention = 7200 // 2 hours

type cached struct {
	timestamp int64
	listed    bool
}

var cache = util.SafeMap[cached]{}
var blacklists []db.Blacklist

func Init() error {
	cache.Init()
	v, err := db.Blacklist{}.GetAll()
	if err != nil {
		return err
	}
	blacklists = v
	return nil
}

func IsListedOn(ip string, blacklist string) bool {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return false
	}
	host := parts[3] + "." + parts[2] + "." + parts[1] + "." + parts[0] +
		"." + blacklist
	v, ok := cache.Get(host)
	if ok && time.Now().Unix()-v.timestamp < retention {
		return v.listed
	}
	ips, err := net.LookupIP(host)
	listed := err == nil && len(ips) > 0
	cache.Set(host, cached{
		listed:    listed,
		timestamp: time.Now().Unix(),
	})
	return listed
}

func IsListed(ip string, readOperation bool) error {
	for _, v := range blacklists {
		if v.Disabled || (v.AllowRead && readOperation) {
			continue
		}
		if IsListedOn(ip, v.Host) {
			return errors.New(ip + " is blacklisted")
		}
	}
	return nil
}

func ClearCache() error {
	cache.Clear()
	v, err := db.Blacklist{}.GetAll()
	if err != nil {
		return err
	}
	blacklists = v
	return err
}
