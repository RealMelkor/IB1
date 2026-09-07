package main

import (
	"fmt"

	"ib1/web"
)

func main() {
	fmt.Println("initialize")
	if err := web.Listen(":8080"); err != nil {
		fmt.Println(err)
	}
}
