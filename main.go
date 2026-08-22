package main

import (
	"log"

	"github.com/lasomethingsomething/cli-prototype/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatalf("Failed to execute CLI: %v", err)
	}
}
