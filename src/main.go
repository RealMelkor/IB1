package main

import (
	"log"
	"IB1/web"
	"IB1/db"
	"IB1/config"
)

func main() {

	if err := config.LoadConfig(); err != nil {
		log.Println(err)
		return
        }

	if err := db.Init(); err != nil {
		log.Println(err)
		return
	}

	if err := parseArguments(); err != nil {
		if err.Error() != "" {
			log.Println(err)
		}
		return
	}

	if err := web.Init(); err != nil {
		log.Println(err)
		return
	}
}
