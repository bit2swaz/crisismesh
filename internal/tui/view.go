package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/bit2swaz/crisismesh/internal/store"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"gorm.io/gorm"
)

var (
	// Colors
	colorGreen = lipgloss.Color("2")
	colorBlack = lipgloss.Color("0")
	colorGray  = lipgloss.Color("240")
	colorRed   = lipgloss.Color("196")
	colorWhite = lipgloss.Color("231")

	// Styles
	baseStyle = lipgloss.NewStyle().
			Foreground(colorGreen).
			Background(colorBlack)

	headerStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colorGreen).
			Padding(0, 1).
			Width(80)

	tabStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(colorGray).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(colorGray)

	activeTabStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(colorGreen).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(colorGreen).
			Bold(true)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(colorBlack).
			Background(colorGreen).
			Padding(0, 1)

	alertStyle = lipgloss.NewStyle().
			Background(colorRed).
			Foreground(colorWhite).
			Bold(true)

	flashStyle = lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(colorRed)

	msgFlashStyle = lipgloss.NewStyle().
			Background(colorWhite).
			Foreground(colorBlack).
			Bold(true)
)

func (m model) View() string {
	if !m.ready {
		return "\n  Initializing System..."
	}

	header := m.renderHeader()
	tabs := m.renderTabs()
	content := m.renderContent()
	statusBar := m.renderStatusBar()

	// Calculate heights for layout
	// Header ~ 5 lines
	// Tabs ~ 2 lines
	// Status ~ 1 line
	// Input ~ 1 line

	body := lipgloss.JoinVertical(lipgloss.Left,
		header,
		tabs,
		content,
		statusBar,
		m.textInput.View(),
	)

	if m.flashTick > 0 && m.flashTick%2 == 0 {
		return flashStyle.Render(body)
	}

	if m.help.ShowAll {
		return lipgloss.JoinVertical(lipgloss.Left, body, m.help.View(m.keys))
	}

	return body
}

func (m model) renderHeader() string {
	logo := `
  CRISISMESH
  [OFFLINE]
`
	stats := fmt.Sprintf("NODE: %s\nTIME: %s\nUPTIME: %s",
		m.nodeID[:8],
		time.Now().Format("15:04:05"),
		time.Since(m.startTime).Round(time.Second),
	)

	return headerStyle.Render(
		lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(50).Render(logo),
			lipgloss.NewStyle().Align(lipgloss.Right).Width(30).Render(stats),
		),
	)
}

func (m model) renderTabs() string {
	var tabs []string

	labels := []string{"COMMs", "NETWORK", "GUIDE"}

	for i, label := range labels {
		if m.activeTab == i {
			tabs = append(tabs, activeTabStyle.Render(label))
		} else {
			tabs = append(tabs, tabStyle.Render(label))
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
}

func (m model) renderContent() string {
	switch m.activeTab {
	case TabComms:
		return m.viewport.View()
	case TabNetwork:
		return m.renderNetwork()
	case TabGuide:
		return m.renderGuide()
	default:
		return m.viewport.View()
	}
}

func (m model) renderNetwork() string {
	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(colorGreen)).
		Headers("PEER ID", "NICK", "LAST SEEN").
		Width(80)

	for _, p := range m.peers {
		lastSeen := p.LastSeen.Format("15:04:05")
		t.Row(p.ID[:8], p.Nick, lastSeen)
	}

	return t.Render()
}

func (m model) renderGuide() string {
	guide := `
  COMMAND GUIDE
  -------------
  /connect <ip:port>  - Manually connect to a peer
  /safe               - Broadcast SAFE status (High Priority)
  
  HOW IT WORKS
  ------------
  1. Messages hop between peers.
  2. If a peer is offline, messages are stored.
  3. When they return, messages are delivered.
  
  STATUS
  ------
  ONLINE: Connected to at least 1 peer.
  ISOLATED: No peers visible.
`
	return lipgloss.NewStyle().Padding(1).Render(guide)
}

func (m model) renderStatusBar() string {
	var status string
	if len(m.peers) > 0 {
		status = fmt.Sprintf("%s ONLINE (%d)", m.spinner.View(), len(m.peers))
	} else {
		status = "❌ ISOLATED"
	}

	left := fmt.Sprintf("STATUS: %s", status)
	right := "TAB: Switch | ?: Help"

	w := m.viewport.Width
	if w == 0 {
		w = 80
	}

	return statusBarStyle.Width(w).Render(
		lipgloss.JoinHorizontal(lipgloss.Center,
			lipgloss.NewStyle().Width(w/2).Align(lipgloss.Left).Render(left),
			lipgloss.NewStyle().Width(w/2).Align(lipgloss.Right).Render(right),
		),
	)
}

func buildChatHistory(db *gorm.DB, nodeID string) (string, error) {
	var sb strings.Builder
	sb.WriteString("Welcome to CrisisMesh!\nChat history will appear here.\n")
	msgs, err := store.GetMessages(db, 50)
	if err != nil {
		return "", err
	}
	for i := len(msgs) - 1; i >= 0; i-- {
		msg := msgs[i]
		sender := "Peer"
		if msg.SenderID == nodeID {
			sender = "You"
		}
		prefix := ""
		if msg.RecipientID == "BROADCAST" {
			prefix = "📢 "
		}
		content := msg.Content
		line := fmt.Sprintf("%s%s: %s", prefix, sender, content)

		if ShouldFlash(time.Unix(msg.Timestamp, 0)) {
			line = msgFlashStyle.Render(line)
		} else if msg.Priority == 2 {
			line = alertStyle.Render(line)
		}
		sb.WriteString(line + "\n")
	}
	return sb.String(), nil
}

func ShouldFlash(msgTime time.Time) bool {
	return time.Since(msgTime) < 500*time.Millisecond
}
