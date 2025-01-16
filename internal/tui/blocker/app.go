package blocker

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
	client       *rpc.Client
	app          *tview.Application
	inbound      *tview.TextView
	outbound     *tview.TextView
	blacklist    *tview.TextView
	statusBar    *tview.TextView
	pages        *tview.Pages
	currentFocus int
	isPaused     bool
}

func NewApp(client *rpc.Client) *App {
	app := &App{
		client:   client,
		app:      tview.NewApplication(),
		isPaused: false,
	}

	app.setupUI()
	return app
}

func (a *App) setupUI() {
	a.pages = tview.NewPages()

	// Create views
	a.inbound = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetScrollable(true)
	a.inbound.SetTitle(" Inbound Connections ").SetBorder(true)

	a.outbound = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetScrollable(true)
	a.outbound.SetTitle(" Outbound Connections ").SetBorder(true)

	a.blacklist = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetScrollable(true)
	a.blacklist.SetTitle(" Black IPs ").SetBorder(true)

	a.statusBar = tview.NewTextView().
		SetDynamicColors(true)

	// Create layout
	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tview.NewFlex().
			AddItem(a.inbound, 0, 1, false).
			AddItem(a.outbound, 0, 1, false),
			0, 2, false).
		AddItem(a.blacklist, 0, 1, false).
		AddItem(a.statusBar, 1, 1, false)

	a.pages.AddPage("main", flex, true, true)
	a.app.SetRoot(a.pages, true)
	a.setupKeyBindings()
}

func (a *App) Run(ctx context.Context) error {
	// Start update goroutine
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if !a.isPaused {
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
			a.statusBar.SetText(fmt.Sprintf("[red]Error: %v", err))
		})
		return
	}

	a.app.QueueUpdateDraw(func() {
		a.updateInboundView(stats.Connections)
		a.updateOutboundView(stats.Connections)
		a.updateBlacklistView()
		a.updateStatusBar()
	})
}

func (a *App) updateInboundView(connections []*models.NewConnectionStats) {
	a.inbound.Clear()
	fmt.Fprintf(a.inbound, "[yellow]%-12s %-25s %-25s[-]\n",
		"Time", "Remote", "Local")

	sort.Slice(connections, func(i, j int) bool {
		return connections[i].Timestamp.After(connections[j].Timestamp)
	})

	for _, conn := range connections {
		if conn.Direction != models.DirectionInbound {
			continue
		}

		fmt.Fprintf(a.inbound, "%-12s %-25s %-25s\n",
			conn.Timestamp.Format("15:04:05"),
			fmt.Sprintf("%s:%d", conn.SrcIP, conn.SrcPort),
			fmt.Sprintf("%s:%d", conn.DstIP, conn.DstPort))
	}
}

func (a *App) updateOutboundView(connections []*models.NewConnectionStats) {
	a.outbound.Clear()
	fmt.Fprintf(a.outbound, "[yellow]%-12s %-25s %-25s[-]\n",
		"Time", "Local", "Remote")

	sort.Slice(connections, func(i, j int) bool {
		return connections[i].Timestamp.After(connections[j].Timestamp)
	})

	for _, conn := range connections {
		if conn.Direction != models.DirectionOutbound {
			continue
		}

		fmt.Fprintf(a.outbound, "%-12s %-25s %-25s\n",
			conn.Timestamp.Format("15:04:05"),
			fmt.Sprintf("%s:%d", conn.SrcIP, conn.SrcPort),
			fmt.Sprintf("%s:%d", conn.DstIP, conn.DstPort))
	}
}

func (a *App) updateBlacklistView() {
	a.blacklist.Clear()
	fmt.Fprintf(a.blacklist, "[yellow]%-20s %-30s %-20s %-40s %-10s[-]\n",
		"Time", "IP Address", "Country", "Reason", "IsBlocked")

	// Get blacklist from client
	stats, err := a.client.GetBlackStats()
	if err != nil {
		fmt.Fprintf(a.blacklist, "[red]Error getting blacklist: %v[-]\n", err)
		return
	}

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Time.After(stats[j].Time)
	})

	for _, stat := range stats {
		fmt.Fprintf(a.blacklist, "%-20s %-30s %-20s %-40s %-10t\n",
			stat.Time.Format("15:04:05"),
			stat.IP,
			stat.Country,
			stat.Reason,
			stat.IsBlocked)
	}
}

func (a *App) updateStatusBar() {
	now := time.Now().Format("2006-01-02 15:04:05")
	pauseStatus := ""
	if a.isPaused {
		pauseStatus = "[red](PAUSED) "
	}
	controls := []string{
		"[yellow]ESC[white]: Quit",
		"[yellow]Tab[white]: Switch View",
		"[yellow]Ctrl+R[white]: Refresh",
		"[yellow]Space[white]: Toggle Pause",
	}
	a.statusBar.SetText(fmt.Sprintf(
		"[white]%sLast updated: %s | %s",
		pauseStatus,
		now,
		strings.Join(controls, " | "),
	))
}

func (a *App) setupKeyBindings() {
	a.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			if a.pages.HasPage("modal") {
				a.pages.RemovePage("modal")
				return nil
			}
			a.app.Stop()
			return nil
		case tcell.KeyCtrlR:
			go a.update()
			return nil
		case tcell.KeyTab:
			a.currentFocus = (a.currentFocus + 1) % 3
			switch a.currentFocus {
			case 0:
				a.app.SetFocus(a.inbound)
			case 1:
				a.app.SetFocus(a.outbound)
			case 2:
				a.app.SetFocus(a.blacklist)
			}
			return nil
		case tcell.KeyRune:
			if event.Rune() == ' ' {
				a.isPaused = !a.isPaused
				a.updateStatusBar()
				return nil
			}
		}
		return event
	})
}
