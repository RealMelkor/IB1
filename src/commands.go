package main

import (
	"os"
	"errors"
	"fmt"
	"syscall"
	"golang.org/x/term"

	"IB1/db"
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
	case "register":
		if len(os.Args) <= 3 {
			return errors.New(os.Args[0] + " register <name> " +
				"<trusted|moderator|admin>")
		}
		rank, err := db.StringToRank(os.Args[3])
		if err != nil { return err }
		password, err := askPassword()
		if err != nil { return err }
		err = db.CreateAccount(os.Args[2], password, rank)
		if err != nil { return err }
	default:
		return errors.New("unknown command: " + os.Args[1])
	}
	return errors.New("")
}
