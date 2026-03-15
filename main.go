package main

import (
	"autoshell/cli"
	"fmt"
	"os"
)

func main() {
	if err := cli.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
