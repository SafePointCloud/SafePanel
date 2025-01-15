package blocker

import (
	"fmt"
	"os/exec"
	"sync"
	"time"
)

// IPBlocker defines the behavior of the IP blocker
type IPBlocker interface {
	Block(ip string, duration time.Duration) error
	Unblock(ip string) error
	IsBlocked(ip string) bool
	GetBlockList() ([]string, error)
}

type BlockRecord struct {
	IP        string
	StartTime time.Time
	Duration  time.Duration
	Reason    string
}

type ipBlocker struct {
	blocked map[string]*BlockRecord
	mutex   sync.RWMutex
	config  *BlockerConfig
}

type BlockerConfig struct {
	IPTables   bool     // Whether to use iptables
	NFTables   bool     // Whether to use nftables
	Whitelist  []string // Whitelist
	DefaultTTL time.Duration
}

func NewIPBlocker(config *BlockerConfig) IPBlocker {
	blocker := &ipBlocker{
		blocked: make(map[string]*BlockRecord),
		config:  config,
	}

	// Start goroutine to clean up expired records
	go blocker.cleanupExpired()

	return blocker
}

func (b *ipBlocker) Block(ip string, duration time.Duration) error {
	if b.isWhitelisted(ip) {
		return fmt.Errorf("IP %s is whitelisted", ip)
	}

	b.mutex.Lock()
	defer b.mutex.Unlock()

	// If already blocked, update duration
	if record, exists := b.blocked[ip]; exists {
		record.Duration = duration
		record.StartTime = time.Now()
		return nil
	}

	// Add firewall rules
	if err := b.addFirewallRules(ip); err != nil {
		return fmt.Errorf("failed to add firewall rules: %v", err)
	}

	// Record block information
	b.blocked[ip] = &BlockRecord{
		IP:        ip,
		StartTime: time.Now(),
		Duration:  duration,
	}

	return nil
}

func (b *ipBlocker) Unblock(ip string) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if err := b.removeIPTablesRule(ip); err != nil {
		return fmt.Errorf("failed to remove iptables rule: %v", err)
	}

	delete(b.blocked, ip)
	return nil
}

func (b *ipBlocker) IsBlocked(ip string) bool {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	record, exists := b.blocked[ip]
	if !exists {
		return false
	}

	// Check if expired
	if record.Duration > 0 && time.Since(record.StartTime) > record.Duration {
		return false
	}

	return true
}

func (b *ipBlocker) GetBlockList() ([]string, error) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	ips := make([]string, 0, len(b.blocked))
	for ip := range b.blocked {
		ips = append(ips, ip)
	}
	return ips, nil
}

func (b *ipBlocker) addIPTablesRule(ip string) error {
	cmd := exec.Command("iptables", "-A", "INPUT", "-s", ip, "-j", "DROP")
	return cmd.Run()
}

func (b *ipBlocker) removeIPTablesRule(ip string) error {
	cmd := exec.Command("iptables", "-D", "INPUT", "-s", ip, "-j", "DROP")
	return cmd.Run()
}

func (b *ipBlocker) cleanupExpired() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		b.mutex.Lock()
		for ip, record := range b.blocked {
			if record.Duration > 0 && time.Since(record.StartTime) > record.Duration {
				b.removeIPTablesRule(ip)
				delete(b.blocked, ip)
			}
		}
		b.mutex.Unlock()
	}
}

func (b *ipBlocker) addFirewallRules(ip string) error {
	var errs []error

	if b.config.IPTables {
		if err := b.addIPTablesRule(ip); err != nil {
			errs = append(errs, err)
		}
	}

	if b.config.NFTables {
		if err := b.addNFTablesRule(ip); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to add firewall rules: %v", errs)
	}

	return nil
}

func (b *ipBlocker) addNFTablesRule(ip string) error {
	cmd := exec.Command("nft", "add", "rule", "ip", "filter", "input", "ip", "saddr", ip, "drop")
	return cmd.Run()
}

func (b *ipBlocker) isWhitelisted(ip string) bool {
	for _, whitelistedIP := range b.config.Whitelist {
		if ip == whitelistedIP {
			return true
		}
	}
	return false
}
