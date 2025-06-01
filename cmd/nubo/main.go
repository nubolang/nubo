package main

import (
	"log"

	"github.com/nubolang/nubo/cmd/nubo/commands"
	"go.uber.org/zap"
)

func main() {
	defer zap.L().Sync()

	err := commands.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
