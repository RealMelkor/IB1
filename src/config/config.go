package config

import "encoding/json"

var Cfg Config

type Config struct {
	Home struct {
		Title		string	`validate:"required"`
		Description	string	`validate:"required"`
		Language	string	`validate:"required"`
		Theme		string	`default:"default"`
	}
	Web struct {
		Domain		string	`default:"localhost"`
		Listener	string	`validate:"required"`
	}
	Media struct {
		Directory	string	`validate:"required"`
		Thumbnail	string	`validate:"required"`
		InDatabase	bool
	}
	Database struct {
                Type            string	`validate:"required"`
                Url             string	`validate:"required"`
        }
	Captcha struct {
		Enabled		bool
		Length		int `validate:"required"`
	}
	Post struct {
		DefaultName	string `default:"Anonymous"`
	}
	Board struct {
		MaxThreads	int `validate:"required"`
	}
}

func LoadDefault() {
	Cfg.Home.Title = "IB1"
	Cfg.Home.Description = "An imageboard that does not require Javascript."
	Cfg.Home.Language = "en"
	Cfg.Home.Theme = "default"
	Cfg.Web.Domain = "localhost"
	Cfg.Web.Listener = ":8080"
	Cfg.Database.Type = "sqlite"
	Cfg.Database.Url = "ib1.db"
	Cfg.Captcha.Enabled = true
	Cfg.Captcha.Length = 7
	Cfg.Board.MaxThreads = 40
	Cfg.Media.Directory = "./media"
	Cfg.Media.Thumbnail = "./thumbnail"
	Cfg.Media.InDatabase = true
}

func LoadConfig(data []byte) error {
        return json.Unmarshal(data, &Cfg)
}

func GetRaw() ([]byte, error) {
	return json.Marshal(&Cfg)
}
