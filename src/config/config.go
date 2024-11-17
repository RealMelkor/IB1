package config

import (
	"encoding/json"
	"crypto/rand"
)

var Cfg Config

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
	}
	Captcha struct {
		Enabled		bool
		Length		int
	}
	Post struct {
		DefaultName	string
		AsciiOnly	bool
		ReadOnly	bool
	}
	Board struct {
		MaxThreads	uint
	}
	Accounts struct {
		AllowRegistration	bool
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
	Cfg.Post.DefaultName = "Anonymous"
	Cfg.Post.AsciiOnly = false
	Cfg.Board.MaxThreads = 40
}

func LoadConfig(data []byte) error {
	if err := json.Unmarshal(data, &Cfg); err != nil { return err }
	if Cfg.Media.Key != nil { return nil }
	Cfg.Media.Key = make([]byte, 64)
	_, err := rand.Read(Cfg.Media.Key)
	return err
}

func GetRaw() ([]byte, error) {
	return json.Marshal(&Cfg)
}
