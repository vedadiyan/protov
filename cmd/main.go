package main

import (
	"fmt"
	"os"

	flaggy "github.com/vedadiyan/flaggy/pkg"
)

func main() {
	args := new(Options)
	if err := flaggy.Parse(args, os.Args[1:]); err != nil {
		fmt.Println(err)
	}

}
