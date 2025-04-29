package config

import (
	"encoding/json"
	"crypto/rand"
)

var Cfg Config

type RateLimit struct {
	MaxAttempts	int
	Timeout		int
}

type Config struct {
	Home struct {
		Title		string
		Description	string
		Language	string
		Theme		string
		Favicon		[]byte
		FaviconMime	string
	}
	Web struct {
		Domain		string
		Listener	string
	}
	Acme struct {
		Email           string
		Port		string
		DisableWWW	bool
	}
	SSL struct {
		Enabled		bool
		Certificate	[]byte
		Key		[]byte
		Listener	string
		DisableHTTP	bool
		RedirectToSSL	bool
	}
	Media struct {
		InDatabase	bool
		Path		string
		Tmp		string
		MaxSize		uint64
		ApprovalQueue	bool
		AllowVideos	bool
		Key		[]byte
		PendingMedia	[]byte
		PendingMime	string
		Spoiler		[]byte
		SpoilerMime	string
		ImageThreshold	int
		HotlinkShield	int
		HotlinkKey	[]byte
	}
	Captcha struct {
		Enabled		bool
		Length		int
	}
	Post struct {
		DefaultName	string
		AsciiOnly	bool
		ReadOnly	bool
		Key		[]byte
	}
	Board struct {
		MaxThreads	uint
	}
	Accounts struct {
		AllowRegistration	bool
		DefaultRank		string
	}
	RateLimit struct {
		Login		RateLimit
		Registration	RateLimit
		Account		RateLimit
		Thread		RateLimit
		Post		RateLimit
	}
}

func LoadDefault() {
	Cfg.Home.Title = "IB1"
	Cfg.Home.Description = "An imageboard that does not require Javascript."
	Cfg.Home.Language = "en"
	Cfg.Home.Theme = "default"
	Cfg.Web.Domain = "localhost"
	Cfg.Web.Listener = ":8080"
	Cfg.SSL.Listener = ":8443"
	Cfg.Captcha.Enabled = true
	Cfg.Captcha.Length = 7
	Cfg.Board.MaxThreads = 40
	Cfg.Media.MaxSize = 1024 * 1024 * 4
	Cfg.Media.InDatabase = true
	Cfg.Media.Path = "./media"
	Cfg.Media.Tmp = "/tmp/ib1"
	Cfg.Media.ImageThreshold = 16
	Cfg.Post.DefaultName = "Anonymous"
	Cfg.Post.AsciiOnly = false
	Cfg.Board.MaxThreads = 40
	Cfg.RateLimit.Login.MaxAttempts = 5
	Cfg.RateLimit.Login.Timeout = 30
	Cfg.RateLimit.Registration.MaxAttempts = 5
	Cfg.RateLimit.Registration.Timeout = 60
	Cfg.RateLimit.Account.MaxAttempts = 30
	Cfg.RateLimit.Account.Timeout = 300
	Cfg.RateLimit.Post.MaxAttempts = 5
	Cfg.RateLimit.Post.Timeout = 60
	Cfg.RateLimit.Thread.MaxAttempts = 2
	Cfg.RateLimit.Thread.Timeout = 120
}

func LoadConfig(data []byte) error {
	if err := json.Unmarshal(data, &Cfg); err != nil { return err }
	if Cfg.Media.Key == nil {
		Cfg.Media.Key = make([]byte, 64)
		_, err := rand.Read(Cfg.Media.Key)
		if err != nil { return err }
	}
	if Cfg.Post.Key == nil {
		Cfg.Post.Key = make([]byte, 512)
		_, err := rand.Read(Cfg.Post.Key)
		if err != nil { return err }
	}
	if Cfg.Media.HotlinkKey == nil {
		Cfg.Media.HotlinkKey = make([]byte, 32)
		_, err := rand.Read(Cfg.Media.HotlinkKey)
		if err != nil { return err }
	}
	return nil
}

func GetRaw() ([]byte, error) {
	return json.Marshal(&Cfg)
}
