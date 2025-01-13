package rpc

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"sync"

	"github.com/safepointcloud/safepanel/internal/analyzer/network"
	"github.com/safepointcloud/safepanel/pkg/models"
)

type StatsServer struct {
	manager    *network.AnalyzerManager
	listener   net.Listener
	clients    map[net.Conn]struct{}
	clientsMux sync.RWMutex
	done       chan struct{}
}

type Stats struct {
	Connections []*models.NewConnectionStats
	DNSQueries  []*models.DNSQueryStats
	IPStats     []*models.ConnectionWindowStats
	PortStats   []*models.PortWindowStats
}

func NewStatsServer(manager *network.AnalyzerManager) *StatsServer {
	return &StatsServer{
		manager: manager,
		clients: make(map[net.Conn]struct{}),
		done:    make(chan struct{}),
	}
}

func (s *StatsServer) Start(socketPath string) error {
	// If the socket file exists, delete it
	if err := os.RemoveAll(socketPath); err != nil {
		return fmt.Errorf("failed to remove existing socket: %v", err)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to create unix socket: %v", err)
	}

	s.listener = listener

	// Start the goroutine to accept connections
	go s.acceptConnections()

	return nil
}

func (s *StatsServer) acceptConnections() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
				log.Printf("Failed to accept connection: %v", err)
				continue
			}
		}

		s.clientsMux.Lock()
		s.clients[conn] = struct{}{}
		s.clientsMux.Unlock()

		go s.handleConnection(conn)
	}
}

func (s *StatsServer) handleConnection(conn net.Conn) {
	defer func() {
		conn.Close()
		s.clientsMux.Lock()
		delete(s.clients, conn)
		s.clientsMux.Unlock()
	}()

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	for {
		var cmd struct {
			Command string         `json:"command"`
			Params  map[string]any `json:"params,omitempty"`
		}

		if err := decoder.Decode(&cmd); err != nil {
			log.Printf("Error decoding command: %v", err)
			return
		}

		var response struct {
			Error string      `json:"error,omitempty"`
			Stats interface{} `json:"stats,omitempty"`
		}

		switch cmd.Command {
		case "GET_STATS":
			stats, err := s.handleGetStats()
			if err != nil {
				response.Error = err.Error()
			} else {
				response.Stats = stats
			}
		case "GET_BLOCK_LIST":
			response.Error = fmt.Sprintf("unknown command: %s", cmd.Command)
		case "GET_BLACK_STATS":
			stats, err := s.handleGetBlackStats()
			if err != nil {
				response.Error = err.Error()
			} else {
				response.Stats = stats
			}
		case "BLOCK_IP":
			response.Error = fmt.Sprintf("unknown command: %s", cmd.Command)
		case "UNBLOCK_IP":
			response.Error = fmt.Sprintf("unknown command: %s", cmd.Command)
		default:
			response.Error = fmt.Sprintf("unknown command: %s", cmd.Command)
		}

		if err := encoder.Encode(response); err != nil {
			log.Printf("Error encoding response: %v", err)
			return
		}
	}
}

func (s *StatsServer) handleGetStats() (*Stats, error) {
	stats := &Stats{}

	// Get the latest 5 minutes of connections
	connections, err := s.manager.GetNewConnections()
	if err != nil {
		log.Printf("Error getting connections: %v", err)
		// Don't return an overall error just because one data fetch failed
		// Continue fetching other data
	}
	stats.Connections = connections

	// Get the latest 5 minutes of DNS queries
	dnsQueries, err := s.manager.GetDNSQueries()
	if err != nil {
		log.Printf("Error getting DNS queries: %v", err)
	}
	stats.DNSQueries = dnsQueries

	// Get IP stats
	ipStats, err := s.manager.GetConnectionWindowStats()
	if err != nil {
		log.Printf("Error getting IP stats: %v", err)
	}
	stats.IPStats = ipStats

	// Get port stats
	portStats, err := s.manager.GetPortWindowStats()
	if err != nil {
		log.Printf("Error getting port stats: %v", err)
	}
	stats.PortStats = portStats

	// Ensure return empty slices instead of nil
	if stats.Connections == nil {
		stats.Connections = []*models.NewConnectionStats{}
	}
	if stats.DNSQueries == nil {
		stats.DNSQueries = []*models.DNSQueryStats{}
	}
	if stats.IPStats == nil {
		stats.IPStats = []*models.ConnectionWindowStats{}
	}
	if stats.PortStats == nil {
		stats.PortStats = []*models.PortWindowStats{}
	}

	return stats, nil
}

func (s *StatsServer) handleGetBlackStats() ([]*models.IPCheckResult, error) {
	return s.manager.GetBlackStats(), nil
}

func (s *StatsServer) Stop() error {
	close(s.done)

	if s.listener != nil {
		s.listener.Close()
	}

	s.clientsMux.Lock()
	for conn := range s.clients {
		conn.Close()
	}
	s.clientsMux.Unlock()

	return nil
}
