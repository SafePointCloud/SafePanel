package network

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/safepointcloud/safepanel/pkg/ipdb"
	"github.com/safepointcloud/safepanel/pkg/mmdb"
	"github.com/safepointcloud/safepanel/pkg/models"
)

// IPChecker Define the behavior of IP checker
type IPChecker interface {
	CheckAndAddToBlacklist(ip string)
	AddToStats(ip string, reason string)
	GetStats() []*models.IPCheckResult
}

type ipChecker struct {
	ipdb         *ipdb.IPDB
	mmdb         *mmdb.MMDB
	checkResults []*models.IPCheckResult
	currentIndex int
	isFull       bool
	mutex        sync.RWMutex
	maxResults   int
	logFile      *os.File
}

func NewIPChecker(ipdb *ipdb.IPDB, mmdb *mmdb.MMDB) IPChecker {
	// Ensure the log directory exists
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		fmt.Printf("Failed to create log directory: %v\n", err)
	}

	// Open the log file
	logPath := filepath.Join(logDir, "ip_checker.log")
	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Printf("Failed to open log file: %v\n", err)
	}

	checker := &ipChecker{
		ipdb:         ipdb,
		mmdb:         mmdb,
		maxResults:   100,
		checkResults: make([]*models.IPCheckResult, 100),
		logFile:      logFile,
	}
	return checker
}

func (c *ipChecker) writeLog(message string) {
	if c.logFile != nil {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		logMessage := fmt.Sprintf("[%s] %s\n", timestamp, message)
		if _, err := c.logFile.WriteString(logMessage); err != nil {
			fmt.Printf("Failed to write to log file: %v\n", err)
		}
	}
}

func (c *ipChecker) CheckAndAddToBlacklist(ip string) {
	if c.ipdb == nil {
		return
	}
	result := c.ipdb.Get([]byte(ip))

	switch result {
	case 1:
		c.AddToStats(ip, "LIGHT Malicious")
	case 2:
		c.AddToStats(ip, "MEDIUM Malicious")
	case 3:
		c.AddToStats(ip, "CRITICAL Malicious")
	default:
	}
}

func (c *ipChecker) AddToStats(ip string, reason string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	result := &models.IPCheckResult{
		IP:        ip,
		IsBlocked: false,
		Reason:    reason,
		Time:      time.Now(),
	}

	info, err := c.mmdb.Lookup(ip)
	if err == nil {
		result.Country = info.RegisteredCountry.Names.En
	}
	c.checkResults[c.currentIndex] = result
	c.currentIndex = (c.currentIndex + 1) % c.maxResults
	if c.currentIndex == 0 {
		c.isFull = true
	}

	c.writeLog(fmt.Sprintf("IP: %s, Country: %s, Reason: %s", ip, result.Country, reason))
}

func (c *ipChecker) GetStats() []*models.IPCheckResult {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	var size int
	if c.isFull {
		size = c.maxResults
	} else {
		size = c.currentIndex
	}

	stats := make([]*models.IPCheckResult, size)
	for i := 0; i < size; i++ {
		idx := (c.currentIndex - size + i + c.maxResults) % c.maxResults
		stats[i] = c.checkResults[idx]
	}
	return stats
}
