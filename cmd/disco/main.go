package main

import (
	"os"

	"github.com/offline-lab/disco/cmd/disco/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
