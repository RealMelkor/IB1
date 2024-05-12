package config

import "github.com/kkyr/fig"

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

func LoadConfig() error {
        err := fig.Load(
                &Cfg,
                fig.File("config.yaml"),
                fig.Dirs(".", "/etc/ib1", "/usr/local/etc/ib1"),
        )
        return err
}
