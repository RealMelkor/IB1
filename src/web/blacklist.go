package web

import (
	"net/url"

	"IB1/dnsbl"
	"IB1/db"
)

func createBlacklist(host string, enabled, allowRead bool) error {
	v, err := url.Parse("https://" + host + "/")
	if err != nil {
		return err
	}
	err = db.Blacklist{}.Add(db.Blacklist{
		Disabled:  !enabled,
		Host:      v.Hostname(),
		AllowRead: allowRead,
	})
	if err != nil {
		return err
	}
	dnsbl.ClearCache()
	return nil
}

func deleteBlacklist(id int) error {
	err := db.Blacklist{}.RemoveID(id, db.Blacklist{})
	if err != nil {
		return err
	}
	dnsbl.ClearCache()
	return nil
}

func updateBlacklist(id int, host string, enabled, allowRead bool) error {
	err := db.Blacklist{}.Update(id, db.Blacklist{
		Host: host, Disabled: !enabled, AllowRead: allowRead,
	})
	if err != nil {
		return err
	}
	dnsbl.ClearCache()
	return nil
}
