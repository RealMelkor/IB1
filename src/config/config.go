package config

import "github.com/kkyr/fig"

var Cfg Config

type Config struct {
	Web struct {
		Domain		string `validate:"required"`
	}
	Database struct {
                Type            string `validate:"required"`
                Url             string `validate:"required"`
        }
	Captcha struct {
		Enabled		bool
	}
	Boards []struct {
		Enabled		bool
		Name		string
		Title		string
		Description	string
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
