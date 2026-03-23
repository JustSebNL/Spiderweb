package main

import (
	"os"

	"github.com/JustSebNL/Spiderweb/cmd/spiderweb/internal/spiderweb"
)

func main() {
	cmd := spiderweb.NewSpiderwebCommand()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

