package main

import (
	"os"

	"github.com/nov11/nacos-cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
