package main

import (
	"os"
	"errors"
	"fmt"
	"syscall"
	"bufio"
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

func firstLaunch() error {
	if count, err := db.AccountsCount(); err != nil || count != 0 {
		return err
	}
	fmt.Println("No account detected, creating admin account")
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter Username: ")
	name, _ := reader.ReadString('\n')
	pop := func(s string) (string) {
		if len(s) < 2 { return "" }
		return s[:len(s) - 1]
	}
	name = pop(name)
	if name != "" && name[len(name) - 1] == '\r' { name = pop(name) }
	if len(name) < 1 { return errors.New("invalid username") }

	password, err := askPassword()
	if err != nil { return err }
	if err := db.Init(); err != nil { return err }

	return db.CreateAccount(name, password, db.RANK_ADMIN)
}

func parseArguments() error {
	if s := os.Getenv("IB1_DB_PATH"); s != "" { db.Path = s }
	if s := os.Getenv("IB1_DB_TYPE"); s != "" { db.Type = s }
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
	case "db":
		if len(os.Args) < 3 {
			return errors.New(os.Args[0] + " db <path> [type]")
		}
		if len(os.Args) > 3 { db.Type = os.Args[3] }
		db.Path = os.Args[2]
		return nil
	default:
		fmt.Println(os.Args[0] +
			" register <name> <trusted|moderator|admin>")
		fmt.Println(os.Args[0] + " domain <domain>")
		fmt.Println(os.Args[0] + " db <path> [sqlite|sqlite3|mysql]")
	}
	return errors.New("")
}
