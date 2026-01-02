package main

import (
	"fmt"
	"os"
)

func main() {
	name := "World"
	if len(os.Args) > 1 {
		name = os.Args[1]
	}
	_, err := fmt.Printf("Hello, %s!\n", name)
	if err != nil {
		os.Exit(1)
	}
}
