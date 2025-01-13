package main

import (
	"context"
	"log"

	"github.com/safepointcloud/safepanel/internal/rpc"
	"github.com/safepointcloud/safepanel/internal/tui/stats"
)

func main() {
	client, err := rpc.NewClient("/var/run/safepanel.sock")
	if err != nil {
		log.Fatalf("Failed to connect to safepanel daemon: %v", err)
	}

	app := stats.NewApp(client)
	if err := app.Run(context.Background()); err != nil {
		log.Fatalf("TUI error: %v", err)
	}
}
