package main

import (
	"os"
	"errors"
	"fmt"
	"syscall"
	"golang.org/x/term"

	"IB1/db"
	"IB1/config"
)

func askPassword() (string, error) {
	fmt.Print("Enter Password: ")
	password, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println("")
	return string(password), err
}

func parseArguments() error {
	if len(os.Args) <= 1 { return nil }
	switch os.Args[1] {
	case "domain":
		if len(os.Args) <= 2 {
			return errors.New(os.Args[0] + " domain <domain>")
		}
		if err := db.Init(); err != nil { return err }
		config.LoadDefault()
		config.Cfg.Web.Domain = os.Args[2]
		if err := db.UpdateConfig(); err != nil { return err }
		fmt.Println("domain updated")
	case "register":
		if len(os.Args) <= 3 {
			return errors.New(os.Args[0] + " register <name> " +
				"<trusted|moderator|admin>")
		}
		rank, err := db.StringToRank(os.Args[3])
		if err != nil { return err }
		password, err := askPassword()
		if err != nil { return err }
		if err := db.Init(); err != nil { return err }
		err = db.CreateAccount(os.Args[2], password, rank)
		if err != nil { return err }
		fmt.Println("new user created")
	case "help":
		fmt.Println(os.Args[0] +
			" register <name> <trusted|moderator|admin>")
		fmt.Println(os.Args[0] + " domain <domain>")
		fmt.Println(os.Args[0] + " help")
	default:
		db.Path = os.Args[1]
		if len(os.Args) > 2 { db.Type = os.Args[2] }
		return nil
	}
	return errors.New("")
}
