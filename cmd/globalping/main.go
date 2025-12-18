package main

import (
	"log"
	"os"

	pkgcli "example.com/rbmq-demo/pkg/cli"
	"github.com/alecthomas/kong"
	"github.com/joho/godotenv"
)

var CLI struct {
	Agent pkgcli.AgentCmd `cmd:"agent"`
	Hub   pkgcli.HubCmd   `cmd:"hub"`
}

func main() {
	if _, err := os.Stat(".env"); err == nil {
		log.Println("Loading .env file")
		err := godotenv.Load()
		if err != nil {
			log.Fatal("Error loading .env file")
		}
	}

	ctx := kong.Parse(&CLI)
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}
