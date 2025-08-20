package main

import (
	"IB1/config"
	"IB1/db"
	"IB1/web"
	"log"
)

const VERSION = "v0.6"

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
