package config

import "encoding/json"

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
	Media struct {
		InDatabase	bool
		Path		string
		Tmp		string
		MaxSize		uint64
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
        return json.Unmarshal(data, &Cfg)
}

func GetRaw() ([]byte, error) {
	return json.Marshal(&Cfg)
}
