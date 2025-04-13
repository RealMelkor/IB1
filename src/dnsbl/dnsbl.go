package dnsbl

import (
	"time"
	"net"
	"strings"
	"errors"

	"IB1/db"
)

const retention = 7200

type cached struct {
	timestamp	int64
	listed		bool
}
var cache = map[string]cached{}

func IsListedOn(ip string, blacklist string) bool {
	parts := strings.Split(ip, ".")
	host := parts[3] + "." + parts[2] + "." + parts[1] + "." + parts[0] +
		"." + blacklist
	v, ok := cache[host]
	if ok && time.Now().Unix() - v.timestamp < retention {
		return v.listed
	}
	ips, err := net.LookupIP(host)
	listed := err == nil && len(ips) > 0
	cache[host] = cached{
		listed: listed,
		timestamp: time.Now().Unix(),
	}
	return listed
}

func IsListed(ip string) error {
	blacklists, err := db.Blacklist{}.GetAll()
	if err != nil { return err }
	for _, v := range blacklists {
		if IsListedOn(ip, v.Host) {
			return errors.New(ip + " is blacklisted")
		}
	}
	return nil
}
