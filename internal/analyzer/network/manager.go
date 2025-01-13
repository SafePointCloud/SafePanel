package network

import (
	"context"
	"time"

	"github.com/samber/lo"

	"github.com/safepointcloud/safepanel/internal/blocker"
	"github.com/safepointcloud/safepanel/pkg/models"
)

type AnalyzerManager struct {
	analyzer  IPAnalyzer
	blocker   blocker.IPBlocker
	checker   IPChecker
	collector *models.StatsCollector
}

func NewAnalyzerManager(analyzer IPAnalyzer, blocker blocker.IPBlocker, checker IPChecker) *AnalyzerManager {
	return &AnalyzerManager{
		analyzer:  analyzer,
		blocker:   blocker,
		checker:   checker,
		collector: models.NewStatsCollector(),
	}
}

func (m *AnalyzerManager) Start(ctx context.Context) error {
	// Set new connection callback
	m.analyzer.SetNewConnectionCallback(func(stats *models.NewConnectionStats) {
		m.collector.AddNewConnection(stats)
		go func(stats *models.NewConnectionStats) {
			ip := stats.SrcIP
			if stats.Direction == models.DirectionOutbound {
				ip = stats.DstIP
			}
			m.checker.CheckAndAddToBlacklist(ip)
		}(stats)
	})

	// Set DNS callback
	m.analyzer.SetDNSQueryCallback(func(query *models.DNSQueryStats) {
		m.collector.AddDNSQuery(query)
	})

	// Set DNS response callback
	m.analyzer.SetDNSResponseCallback(func(response *models.DNSResponse) {
		m.collector.AddDNSResponse(response)
	})

	// Start the analyzer
	if err := m.analyzer.Start(ctx); err != nil {
		return err
	}

	go m.runCleanup(ctx)

	return nil
}

func (m *AnalyzerManager) GetNewConnections() ([]*models.NewConnectionStats, error) {
	return m.collector.GetNewConnections(), nil
}

func (m *AnalyzerManager) GetDNSQueries() ([]*models.DNSQueryStats, error) {
	return m.collector.GetDNSQueries(), nil
}

func (m *AnalyzerManager) GetConnectionWindowStats() ([]*models.ConnectionWindowStats, error) {
	return lo.Values(m.collector.GetConnectionWindows()), nil
}

func (m *AnalyzerManager) GetPortWindowStats() ([]*models.PortWindowStats, error) {
	return lo.Values(m.collector.GetPortWindows()), nil
}

// Add cleanup goroutine
func (m *AnalyzerManager) runCleanup(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.collector.CleanupOldStats()
		}
	}
}

func (m *AnalyzerManager) GetBlackStats() []*models.IPCheckResult {
	return m.checker.GetStats()
}
