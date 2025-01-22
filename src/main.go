package main

import (
	"log"
	"IB1/web"
	"IB1/db"
	"IB1/config"
)

const VERSION = "v0.5"

func main() {

	if err := parseArguments(); err != nil {
		if err.Error() != "" {
			log.Println(err)
		}
		return
	}

	if err := db.Init(); err != nil {
		log.Println(err)
		return
	}

	if err := firstLaunch(); err != nil {
		log.Println(err)
		return
	}

	log.Println("IB1", VERSION, "- Listening on", config.Cfg.Web.Listener)
	if err := web.Init(); err != nil {
		log.Println(err)
		return
	}
}
