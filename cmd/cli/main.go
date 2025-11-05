package main

import (
	"os"

	"github.com/davidmoltin/intelligent-workflows/cmd/cli/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
