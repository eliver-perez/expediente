package main

import (
	"fmt"
	"gestor-documental/internal/folderpicker"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		os.Exit(2)
	}
	if err := folderpicker.Run(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
