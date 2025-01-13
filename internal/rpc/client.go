package rpc

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/safepointcloud/safepanel/pkg/models"
)

type Client struct {
	address     string
	conn        net.Conn
	mutex       sync.Mutex
	reconnectCh chan struct{}
}

func NewClient(address string) (*Client, error) {
	client := &Client{
		address:     address,
		reconnectCh: make(chan struct{}, 1),
	}

	// Initial connection
	if err := client.connect(); err != nil {
		return nil, err
	}

	// Start reconnect monitor
	go client.reconnectMonitor()

	return client, nil
}

func (c *Client) connect() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.conn != nil {
		c.conn.Close()
	}

	conn, err := net.Dial("unix", c.address)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %v", c.address, err)
	}

	c.conn = conn
	return nil
}

func (c *Client) reconnectMonitor() {
	for range c.reconnectCh {
		for i := 0; i < 5; i++ { // Try up to 5 times
			if err := c.connect(); err == nil {
				break
			}
			time.Sleep(time.Second * time.Duration(i+1)) // Exponential backoff
		}
	}
}

func (c *Client) GetStats() (*Stats, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	cmd := struct {
		Command string `json:"command"`
	}{
		Command: "GET_STATS",
	}

	if err := json.NewEncoder(c.conn).Encode(cmd); err != nil {
		select {
		case c.reconnectCh <- struct{}{}:
		default:
		}
		return nil, fmt.Errorf("failed to send command: %v", err)
	}

	var response struct {
		Error string `json:"error,omitempty"`
		Stats *Stats `json:"stats,omitempty"`
	}

	if err := json.NewDecoder(c.conn).Decode(&response); err != nil {
		select {
		case c.reconnectCh <- struct{}{}:
		default:
		}
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	if response.Error != "" {
		return nil, fmt.Errorf("server error: %s", response.Error)
	}

	if response.Stats == nil {
		return nil, fmt.Errorf("received nil stats from server")
	}

	// Ensure all slices are initialized
	if response.Stats.Connections == nil {
		response.Stats.Connections = []*models.NewConnectionStats{}
	}
	if response.Stats.DNSQueries == nil {
		response.Stats.DNSQueries = []*models.DNSQueryStats{}
	}
	if response.Stats.IPStats == nil {
		response.Stats.IPStats = []*models.ConnectionWindowStats{}
	}
	if response.Stats.PortStats == nil {
		response.Stats.PortStats = []*models.PortWindowStats{}
	}

	return response.Stats, nil
}

func (c *Client) GetBlackStats() ([]*models.IPCheckResult, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	cmd := struct {
		Command string `json:"command"`
	}{
		Command: "GET_BLACK_STATS",
	}

	if err := json.NewEncoder(c.conn).Encode(cmd); err != nil {
		return nil, fmt.Errorf("failed to send command: %v", err)
	}

	var response struct {
		Error string                  `json:"error,omitempty"`
		Stats []*models.IPCheckResult `json:"stats,omitempty"`
	}

	if err := json.NewDecoder(c.conn).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	return response.Stats, nil
}

func (c *Client) GetBlockedIPs() ([]string, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	cmd := struct {
		Command string `json:"command"`
	}{
		Command: "GET_BLOCKED_IPS",
	}

	if err := json.NewEncoder(c.conn).Encode(cmd); err != nil {
		return nil, fmt.Errorf("failed to send command: %v", err)
	}

	var response struct {
		Error string   `json:"error,omitempty"`
		IPs   []string `json:"ips,omitempty"`
	}

	if err := json.NewDecoder(c.conn).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	return response.IPs, nil
}

func (c *Client) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
