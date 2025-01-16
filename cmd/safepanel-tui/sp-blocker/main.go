package main

import (
	"context"
	"log"

	"github.com/safepointcloud/safepanel/internal/rpc"
	"github.com/safepointcloud/safepanel/internal/tui/blocker"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	log.Printf("SafePanel Blocker %s (%s) built at %s", version, commit, date)

	client, err := rpc.NewClient("/var/run/safepanel.sock")
	if err != nil {
		log.Fatalf("Failed to connect to safepanel daemon: %v", err)
	}

	app := blocker.NewApp(client)
	if err := app.Run(context.Background()); err != nil {
		log.Fatalf("TUI error: %v", err)
	}
}
