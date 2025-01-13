package models

import (
	"fmt"
	"sync"
	"time"
)

type Direction uint8

const (
	DirectionInbound  Direction = 0
	DirectionOutbound Direction = 1
)

type Protocol string

const (
	ProtocolTCP Protocol = "TCP"
	ProtocolUDP Protocol = "UDP"
)

// NewConnectionStats represents individual new connection events
type NewConnectionStats struct {
	SrcIP     string
	SrcPort   uint16
	DstIP     string
	DstPort   uint16
	Protocol  Protocol
	Direction Direction
	Timestamp time.Time
}

// DNSQueryStats represents DNS query statistics
type DNSQueryStats struct {
	ID        uint16
	Domain    string
	SrcIP     string
	Response  []string
	DNSServer string
	QueryType string
	Timestamp time.Time
}

type DNSResponse struct {
	QueryID   uint16
	Response  []string
	Timestamp time.Time
}

// ConnectionWindowStats represents 10-minute window statistics per source IP->dest IP pair
type ConnectionWindowStats struct {
	SrcIP       string
	DstIP       string
	Ports       map[uint16]int // port -> count
	TotalConns  int
	WindowStart time.Time
	WindowEnd   time.Time
}

// PortWindowStats represents 10-minute window statistics per destination port
type PortWindowStats struct {
	DstIP       string
	DstPort     uint16
	UniqueIPs   map[string]struct{} // set of source IPs
	TotalConns  int64
	WindowStart time.Time
	WindowEnd   time.Time
}

type IPCheckResult struct {
	IP        string
	IsBlocked bool
	Reason    string
	Country   string
	ASN       string
	Time      time.Time
}

const maxRecords = 1000

// StatsCollector handles the collection and aggregation of all statistics
type StatsCollector struct {
	NewConnections    []*NewConnectionStats
	DNSQueries        []*DNSQueryStats
	ConnectionWindows map[string]*ConnectionWindowStats
	PortWindows       map[string]*PortWindowStats
	windowDuration    time.Duration
	mutex             sync.RWMutex
	maxResults        int
	connIndex         int
	dnsIndex          int
	connIsFull        bool
	dnsIsFull         bool
}

func NewStatsCollector() *StatsCollector {
	return &StatsCollector{
		NewConnections:    make([]*NewConnectionStats, maxRecords),
		DNSQueries:        make([]*DNSQueryStats, maxRecords),
		ConnectionWindows: make(map[string]*ConnectionWindowStats),
		PortWindows:       make(map[string]*PortWindowStats),
		windowDuration:    10 * time.Minute,
		maxResults:        maxRecords,
	}
}

func (sc *StatsCollector) AddNewConnection(stats *NewConnectionStats) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	// add new connection using circular index
	sc.NewConnections[sc.connIndex] = stats
	sc.connIndex = (sc.connIndex + 1) % sc.maxResults
	if sc.connIndex == 0 {
		sc.connIsFull = true
	}

	// update connection window stats
	key := fmt.Sprintf("%s->%s", stats.SrcIP, stats.DstIP)
	if cw, exists := sc.ConnectionWindows[key]; exists {
		cw.AddPort(stats.DstPort)
		cw.WindowEnd = time.Now()
	} else {
		cw = NewConnectionWindowStats(stats.SrcIP, stats.DstIP)
		cw.AddPort(stats.DstPort)
		sc.ConnectionWindows[key] = cw
	}

	// update port window stats
	portKey := fmt.Sprintf("%s:%d", stats.DstIP, stats.DstPort)
	if pw, exists := sc.PortWindows[portKey]; exists {
		pw.TotalConns++
		pw.UniqueIPs[stats.SrcIP] = struct{}{}
		pw.WindowEnd = time.Now()
	} else {
		sc.PortWindows[portKey] = &PortWindowStats{
			DstIP:       stats.DstIP,
			DstPort:     stats.DstPort,
			UniqueIPs:   map[string]struct{}{stats.SrcIP: {}},
			TotalConns:  1,
			WindowStart: time.Now(),
			WindowEnd:   time.Now(),
		}
	}
}

func (sc *StatsCollector) AddDNSQuery(stats *DNSQueryStats) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	// add new DNS query using circular index
	sc.DNSQueries[sc.dnsIndex] = stats
	sc.dnsIndex = (sc.dnsIndex + 1) % sc.maxResults
	if sc.dnsIndex == 0 {
		sc.dnsIsFull = true
	}
}

func (c *StatsCollector) AddDNSResponse(stats *DNSResponse) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	query := c.GetMatchedDNSQuery(stats.QueryID)
	if query != nil {
		query.Response = append(query.Response, stats.Response...)
	}
}

// add method to associate query and response in StatsCollector
func (c *StatsCollector) GetMatchedDNSQuery(queryID uint16) *DNSQueryStats {
	// match query and response by ID
	var query *DNSQueryStats

	// find in query list
	for _, q := range c.DNSQueries {
		if q.ID == queryID {
			query = q
			break
		}
	}

	return query
}

func (sc *StatsCollector) CleanupOldStats() {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	threshold := time.Now().Add(-sc.windowDuration)

	// cleanup connection window
	for key, stats := range sc.ConnectionWindows {
		if stats.WindowEnd.Before(threshold) {
			delete(sc.ConnectionWindows, key)
		}
	}

	// cleanup port window
	for key, stats := range sc.PortWindows {
		if stats.WindowEnd.Before(threshold) {
			delete(sc.PortWindows, key)
		}
	}
}

func NewConnectionWindowStats(srcIP, dstIP string) *ConnectionWindowStats {
	return &ConnectionWindowStats{
		SrcIP:       srcIP,
		DstIP:       dstIP,
		Ports:       make(map[uint16]int),
		TotalConns:  0,
		WindowStart: time.Now(),
		WindowEnd:   time.Now(),
	}
}

// AddPort adds a destination port to the statistics
func (cw *ConnectionWindowStats) AddPort(port uint16) {
	cw.Ports[port]++
	cw.TotalConns++
}

// add method to get stats, return copy instead of direct reference
func (sc *StatsCollector) GetConnectionWindows() map[string]*ConnectionWindowStats {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()

	// create a copy
	result := make(map[string]*ConnectionWindowStats, len(sc.ConnectionWindows))
	for k, v := range sc.ConnectionWindows {
		// deep copy ConnectionWindowStats
		newStats := &ConnectionWindowStats{
			SrcIP:       v.SrcIP,
			DstIP:       v.DstIP,
			Ports:       make(map[uint16]int),
			TotalConns:  v.TotalConns,
			WindowStart: v.WindowStart,
			WindowEnd:   v.WindowEnd,
		}
		for port, count := range v.Ports {
			newStats.Ports[port] = count
		}
		result[k] = newStats
	}
	return result
}

func (sc *StatsCollector) GetPortWindows() map[string]*PortWindowStats {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()

	// create a copy
	result := make(map[string]*PortWindowStats, len(sc.PortWindows))
	for k, v := range sc.PortWindows {
		// deep copy PortWindowStats
		newStats := &PortWindowStats{
			DstIP:       v.DstIP,
			DstPort:     v.DstPort,
			UniqueIPs:   make(map[string]struct{}),
			TotalConns:  v.TotalConns,
			WindowStart: v.WindowStart,
			WindowEnd:   v.WindowEnd,
		}
		for ip := range v.UniqueIPs {
			newStats.UniqueIPs[ip] = struct{}{}
		}
		result[k] = newStats
	}
	return result
}

func (sc *StatsCollector) GetNewConnections() []*NewConnectionStats {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()

	var size int
	if sc.connIsFull {
		size = sc.maxResults
	} else {
		size = sc.connIndex
	}

	results := make([]*NewConnectionStats, size)
	for i := 0; i < size; i++ {
		idx := (sc.connIndex - size + i + sc.maxResults) % sc.maxResults
		results[i] = sc.NewConnections[idx]
	}
	return results
}

func (sc *StatsCollector) GetDNSQueries() []*DNSQueryStats {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()

	var size int
	if sc.dnsIsFull {
		size = sc.maxResults
	} else {
		size = sc.dnsIndex
	}

	results := make([]*DNSQueryStats, size)
	for i := 0; i < size; i++ {
		idx := (sc.dnsIndex - size + i + sc.maxResults) % sc.maxResults
		results[i] = sc.DNSQueries[idx]
	}
	return results
}
