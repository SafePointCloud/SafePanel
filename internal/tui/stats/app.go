package stats

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/safepointcloud/safepanel/internal/rpc"
	"github.com/safepointcloud/safepanel/pkg/models"
)

type App struct {
	client      *rpc.Client
	app         *tview.Application
	connections *tview.TextView
	dns         *tview.TextView
	ipStats     *tview.TextView
	portStats   *tview.TextView
	statusBar   *tview.TextView
}

func NewApp(client *rpc.Client) *App {
	app := &App{
		client: client,
		app:    tview.NewApplication(),
	}

	app.setupUI()
	return app
}

func (a *App) setupUI() {
	// create views
	a.connections = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetScrollable(true)

	a.dns = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetScrollable(true)
	a.dns.SetWrap(false)

	a.ipStats = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetScrollable(true)

	a.portStats = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetScrollable(true)

	a.statusBar = tview.NewTextView().
		SetDynamicColors(true)

	// create layout
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tview.NewFlex().
			AddItem(a.connections, 0, 1, false).
			AddItem(a.dns, 0, 1, false),
			0, 2, false).
		AddItem(tview.NewFlex().
			AddItem(a.ipStats, 0, 1, false).
			AddItem(a.portStats, 0, 1, false),
			0, 2, false).
		AddItem(a.statusBar, 1, 1, false)

	a.app.SetRoot(flex, true)
}

func (a *App) Run(ctx context.Context) error {
	if err := a.Init(); err != nil {
		return fmt.Errorf("failed to initialize app: %v", err)
	}

	if a.app == nil || a.connections == nil || a.dns == nil ||
		a.ipStats == nil || a.portStats == nil || a.statusBar == nil {
		return fmt.Errorf("app components not properly initialized")
	}

	// start update goroutine
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		retryTicker := time.NewTicker(time.Second * 5)
		defer retryTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				a.update()
			case <-retryTicker.C:
				if a.statusBar != nil && strings.Contains(a.statusBar.GetText(true), "[red]Error") {
					a.update()
				}
			}
		}
	}()

	return a.app.Run()
}

func (a *App) update() {
	stats, err := a.client.GetStats()
	if err != nil {
		a.app.QueueUpdateDraw(func() {
			if a.statusBar != nil {
				a.statusBar.SetText(fmt.Sprintf("[red]Error: %v - Retrying...", err))
			}
		})
		return
	}

	a.app.QueueUpdateDraw(func() {
		if stats == nil {
			if a.statusBar != nil {
				a.statusBar.SetText("[red]Error: Received nil stats")
			}
			return
		}

		if a.connections != nil {
			a.updateConnectionsView(stats.Connections)
		}
		if a.dns != nil {
			a.updateDNSView(stats.DNSQueries)
		}
		if a.ipStats != nil {
			a.updateIPStatsView(stats.IPStats)
		}
		if a.portStats != nil {
			a.updatePortStatsView(stats.PortStats)
		}
		if a.statusBar != nil {
			a.updateStatusBar()
		}
	})
}

func (a *App) updateConnectionsView(connections []*models.NewConnectionStats) {
	a.connections.Clear()
	fmt.Fprintf(a.connections, "[yellow]%-12s %-25s %-25s[-]\n",
		"Time", "Source", "Destination")

	for _, conn := range connections {
		fmt.Fprintf(a.connections, "%-12s %-25s %-25s\n",
			conn.Timestamp.Format("15:04:05"),
			fmt.Sprintf("%s:%d", conn.SrcIP, conn.SrcPort),
			fmt.Sprintf("%s:%d", conn.DstIP, conn.DstPort))
	}
}

func (a *App) updateDNSView(queries []*models.DNSQueryStats) {
	a.dns.Clear()
	fmt.Fprintf(a.dns, "[yellow]%-12s %-30s %-30s %-10s %-10s[-]\n",
		"Time", "Domain", "Response", "Client", "DNS Server")

	for _, query := range queries {
		fmt.Fprintf(a.dns, "%-12s %-30s %-30s %-10s %-10s\n",
			query.Timestamp.Format("15:04:05"),
			truncateString(query.Domain, 28),
			truncateString(strings.Join(query.Response, ","), 28),
			query.SrcIP,
			query.DNSServer)
	}
}

func (a *App) updateIPStatsView(stats []*models.ConnectionWindowStats) {
	a.ipStats.Clear()
	fmt.Fprintf(a.ipStats, "[yellow]%-30s %-15s %-15s[-]\n",
		"SrcIP -> DstIP", "Unique Ports", "Conns")

	// sort by connection count
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].TotalConns > stats[j].TotalConns
	})

	for _, stat := range stats {
		fmt.Fprintf(a.ipStats, "%-30s %-15d %-15d\n",
			stat.SrcIP+" -> "+stat.DstIP,
			len(stat.Ports),
			stat.TotalConns)
	}
}

func (a *App) updatePortStatsView(stats []*models.PortWindowStats) {
	a.portStats.Clear()
	fmt.Fprintf(a.portStats, "[yellow]%-25s %-15s %-15s[-]\n",
		"DstIP:Port", "Unique IPs", "Conns")

	// sort by connection count
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].TotalConns > stats[j].TotalConns
	})

	for _, stat := range stats {
		fmt.Fprintf(a.portStats, "%-25s %-15d %-15d\n",
			fmt.Sprintf("%s:%d", stat.DstIP, stat.DstPort),
			len(stat.UniqueIPs),
			stat.TotalConns)
	}
}

// truncateString truncate string and add ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// setup key bindings
func (a *App) setupKeyBindings() {
	a.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			a.app.Stop()
			return nil
		case tcell.KeyCtrlR:
			// use non-blocking way to trigger refresh
			go func() {
				a.app.QueueUpdateDraw(func() {
					if a.statusBar != nil {
						a.statusBar.SetText("[yellow]Refreshing...[white]")
					}
				})
				// reuse existing update method
				a.update()
			}()
			return nil
		case tcell.KeyTab:
			// TODO: implement focus switch logic
			return nil
		}
		return event
	})
}

func (a *App) updateStatusBar() {
	now := time.Now().Format("2006-01-02 15:04:05")
	a.statusBar.SetText(fmt.Sprintf("[white]Last updated: %s | Press [yellow]ESC[white] to quit | [yellow]Ctrl+R[white] to refresh", now))
}

func (a *App) Init() error {
	if a.app == nil {
		a.app = tview.NewApplication()
	}

	if a.connections == nil {
		a.connections = tview.NewTextView().
			SetDynamicColors(true).
			SetRegions(true).
			SetScrollable(true)
	}

	if a.dns == nil {
		a.dns = tview.NewTextView().
			SetDynamicColors(true).
			SetRegions(true).
			SetScrollable(true)
	}

	if a.ipStats == nil {
		a.ipStats = tview.NewTextView().
			SetDynamicColors(true).
			SetRegions(true).
			SetScrollable(true)
	}

	if a.portStats == nil {
		a.portStats = tview.NewTextView().
			SetDynamicColors(true).
			SetRegions(true).
			SetScrollable(true)
	}

	if a.statusBar == nil {
		a.statusBar = tview.NewTextView().
			SetDynamicColors(true)
	}

	// set title and border
	a.connections.SetTitle(" Connections ").SetBorder(true)
	a.dns.SetTitle(" DNS Queries ").SetBorder(true)
	a.ipStats.SetTitle(" IP Statistics ").SetBorder(true)
	a.portStats.SetTitle(" Port Statistics ").SetBorder(true)

	// create layout
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tview.NewFlex().
			AddItem(a.connections, 0, 1, false).
			AddItem(a.dns, 0, 1, false),
			0, 2, false).
		AddItem(tview.NewFlex().
			AddItem(a.ipStats, 0, 1, false).
			AddItem(a.portStats, 0, 1, false),
			0, 2, false).
		AddItem(a.statusBar, 1, 1, false)

	// set root layout
	a.app.SetRoot(flex, true)

	// setup key bindings
	a.setupKeyBindings()

	// init status bar
	a.updateStatusBar()

	return nil
}
