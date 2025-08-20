package db

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"io"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/yl2chen/cidranger"
)

var countries = map[string]cidranger.Ranger{}

type CIDR struct {
	CIDR    string
	Country string
}

const ZonesURL = "https://www.ipdeny.com/ipblocks/data/countries/all-zones.tar.gz"

func LoadCountries() error {
	var count int64
	db.Model(&CIDR{}).Count(&count)
	if count == 0 {
		if err := UpdateZones(ZonesURL); err != nil {
			return err
		}
		db.Model(&CIDR{}).Count(&count)
	}
	rows, err := db.Model(&CIDR{}).Rows()
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var v CIDR
		if err := db.ScanRows(rows, &v); err != nil {
			return err
		}
		if v.Country == "zz" {
			continue
		}
		tmp, ok := countries[v.Country]
		if !ok {
			tmp = cidranger.NewPCTrieRanger()
		}
		_, cidr, _ := net.ParseCIDR(v.CIDR)
		if cidr == nil {
			continue
		}
		tmp.Insert(cidranger.NewBasicRangerEntry(*cidr))
		countries[v.Country] = tmp
	}
	log.Println(count, "CIDRs entries loaded")
	return nil
}

func UpdateZones(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	uncompress, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	defer uncompress.Close()
	tarReader := tar.NewReader(uncompress)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		info := header.FileInfo()
		if info.IsDir() {
			continue
		}

		country := strings.Split(info.Name(), ".")[0]
		log.Println("Loading zone: '" + country + "'")
		if loadCountryBlocks(tarReader, country); err != nil {
			return err
		}
	}
	return nil
}

func loadCountryBlocks(reader io.Reader, country string) error {
	if country == "zz" {
		return nil
	}
	buf := bufio.NewReader(reader)
	var CIDRs [1024]CIDR
	for i := 0; ; i++ {
		line, _, err := buf.ReadLine()
		if err == io.EOF {
			if i == 0 {
				return nil
			}
			return db.Create(CIDRs[0:i]).Error
		}
		if err != nil {
			return err
		}
		_, ip, err := net.ParseCIDR(string(line))
		if err != nil {
			return err
		}
		if ip == nil {
			continue
		}

		if i == len(CIDRs) {
			err := db.Create(CIDRs[0:i]).Error
			if err != nil {
				return err
			}
			i = 0
		}
		CIDRs[i] = CIDR{CIDR: ip.String(), Country: country}
	}
}

func GetCountry(_ip string) string {
	ip := net.ParseIP(_ip)
	if ip == nil {
		return ""
	}
	for k, v := range countries {
		b, err := v.Contains(ip)
		if err != nil {
			return ""
		}
		if b {
			return k
		}
	}
	return ""
}
