package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "net/http/pprof"

	"github.com/safepointcloud/safepanel/internal/analyzer/network"
	"github.com/safepointcloud/safepanel/internal/blocker"
	"github.com/safepointcloud/safepanel/internal/config"
	"github.com/safepointcloud/safepanel/internal/rpc"
	"github.com/safepointcloud/safepanel/pkg/ipdb"
	"github.com/safepointcloud/safepanel/pkg/mmdb"
)

func main() {
	// Initialize config
	if err := config.Init(); err != nil {
		log.Fatalf("Failed to initialize config: %v", err)
	}
	cfg := config.Get()

	// if profiling is enabled
	if cfg.Profiling.Enabled {
		go func() {
			port := cfg.Profiling.Port
			log.Printf("Starting pprof server on localhost:%d", port)
			if err := http.ListenAndServe(fmt.Sprintf("localhost:%d", port), nil); err != nil {
				log.Printf("Failed to start pprof server: %v", err)
			}
		}()
	}

	// create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// initialize IP analyzer
	analyzerConfig := &network.Config{
		Interface:   cfg.Analyzer.Network.IP.Interface,
		BufferSize:  cfg.Analyzer.Network.IP.BufferSize,
		Promiscuous: cfg.Analyzer.Network.IP.Promiscuous,
	}

	analyzer, err := network.NewIPAnalyzer(analyzerConfig)
	if err != nil {
		log.Fatalf("Failed to create IP analyzer: %v", err)
	}

	// initialize IP blocker
	blockerConfig := &blocker.BlockerConfig{
		IPTables:   cfg.Blocker.IP.IPTables,
		NFTables:   cfg.Blocker.IP.NFTables,
		Whitelist:  cfg.Blocker.IP.Whitelist,
		DefaultTTL: time.Hour,
	}
	blocker := blocker.NewIPBlocker(blockerConfig)
	ipdb, err := ipdb.NewIPDB(cfg.Checker.IPDBPath)
	if err != nil {
		log.Fatalf("Failed to load IPDB: %v", err)
	}
	mmdb, err := mmdb.NewMMDB(cfg.Checker.MMDBPath)
	if err != nil {
		log.Fatalf("Failed to load MMDB: %v", err)
	}
	checker := network.NewIPChecker(ipdb, mmdb)

	manager := network.NewAnalyzerManager(analyzer, blocker, checker)
	if err := manager.Start(ctx); err != nil {
		log.Fatalf("Failed to start analyzer manager: %v", err)
	}

	// start RPC server
	server := rpc.NewStatsServer(manager)
	if err := server.Start("/var/run/safepanel.sock"); err != nil {
		log.Fatalf("Failed to start RPC server: %v", err)
	}
	defer server.Stop()

	// wait for signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
}
