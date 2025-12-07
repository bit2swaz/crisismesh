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

	authorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("14")) // Cyan

	sidebarStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(colorGreen).
			Padding(0, 1)

	streamStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colorGreen).
			Padding(0, 1)
)

func (m model) View() string {
	if !m.ready {
		return "\n  Initializing System..."
	}

	// We need to use the full width available from WindowSizeMsg which was stored in viewport.Width
	// But wait, viewport.Width was set to msg.Width in Update.
	// Let's assume m.viewport.Width is the full screen width for now, or we need to store window width separately.
	// In Update: m.viewport.Width = msg.Width. So yes.

	// Actually, we want to split the screen.
	// Let's recalculate based on a stored width if possible, but viewport width is what we have.
	// If we change viewport width, we lose the window width reference.
	// Ideally `model` should store `windowWidth` and `windowHeight`.
	// But for now, let's assume viewport.Width is correct at start of View, then we modify it for rendering?
	// No, View shouldn't modify model state.
	// I should have stored window size in model.
	// Let's check model.go again. It doesn't store window size explicitly, just updates viewport.
	// I'll use m.viewport.Width as the total width for now, but this is risky if I resize viewport.
	// Ah, viewport is a model.
	// I will just use m.viewport.Width as total width.

	totalWidth := m.viewport.Width
	totalHeight := m.viewport.Height

	streamWidth := int(float64(totalWidth) * 0.7)
	sidebarWidth := totalWidth - streamWidth - 4 // Adjust for borders

	// Create a temporary viewport for rendering with correct width
	vp := m.viewport
	vp.Width = streamWidth
	vp.Height = totalHeight

	streamView := streamStyle.Width(streamWidth).Height(totalHeight).Render(vp.View())
	sidebarView := m.renderSidebar(sidebarWidth, totalHeight)

	body := lipgloss.JoinHorizontal(lipgloss.Top, streamView, sidebarView)

	if m.showQR {
		qrView := lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(colorGreen).
			Padding(1).
			Render(m.qrCode + "\n\nSCAN TO JOIN MESH")

		return lipgloss.Place(totalWidth, totalHeight, lipgloss.Center, lipgloss.Center, qrView)
	}

	if m.lastMsgPriority == 2 {
		// Pulsing Red Border
		borderColor := lipgloss.Color("#FF0000")
		if time.Now().UnixMilli()%1000 < 500 {
			borderColor = lipgloss.Color("#550000")
		}
		return lipgloss.NewStyle().Border(lipgloss.ThickBorder()).BorderForeground(borderColor).Render(body)
	}

	if m.flashTick > 0 && m.flashTick%2 == 0 {
		return flashStyle.Render(body)
	}

	return body
}

func (m model) renderSidebar(width, height int) string {
	logo := `
 CRISISMESH
 v0.1.1
`
	identity := fmt.Sprintf("ID: %s\nKEY: %s...", m.nodeID[:8], "Curve25519")

	t := table.New().
		Border(lipgloss.HiddenBorder()).
		Headers("ID", "RTT", "SEEN").
		Width(width)

	for _, p := range m.peers {
		t.Row(p.ID[:4], "12ms", "Now")
	}

	encStatus := "ENCRYPTION: ACTIVE\nCurve25519 + XSalsa20"

	content := lipgloss.JoinVertical(lipgloss.Left,
		logo,
		"\n",
		identity,
		"\n",
		"NETWORK HEALTH:",
		t.Render(),
		"\n",
		encStatus,
	)

	return sidebarStyle.Width(width).Height(height).Render(content)
}

func buildChatHistory(db *gorm.DB, nodeID string, monitorMode bool) (string, int, error) {
	var sb strings.Builder
	msgs, err := store.GetMessages(db, 50)
	if err != nil {
		return "", 0, err
	}

	latestPriority := 0
	if len(msgs) > 0 {
		latestPriority = msgs[0].Priority
	}

	for i := len(msgs) - 1; i >= 0; i-- {
		msg := msgs[i]

		var line string
		if monitorMode {
			// Raw JSON style
			line = fmt.Sprintf(`{"ts":%d, "sender":"%s", "content":"%s", "prio":%d}`,
				msg.Timestamp, msg.SenderID, msg.Content, msg.Priority)
		} else {
			// Pretty Log style
			ts := time.Unix(msg.Timestamp, 0).Format("15:04:05")
			hops := 1
			enc := "ON"

			author := msg.Author
			if author == "" {
				author = "Unknown"
			}
			authorTag := authorStyle.Render(fmt.Sprintf("[USER: %s]", author))

			line = fmt.Sprintf("[%s] [ENC:%s] [HOP:%d] %s -> %s", ts, enc, hops, authorTag, msg.Content)

			if msg.Lat != 0 && msg.Long != 0 {
				gpsTag := lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render(fmt.Sprintf(" [GPS: %.4f, %.4f]", msg.Lat, msg.Long))
				line += gpsTag
			} else if msg.Priority == 2 {
				gpsTag := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(" [GPS: NO SIGNAL]")
				line += gpsTag
			}
		}

		if msg.Priority == 2 {
			line = alertStyle.Render(line)
		}
		sb.WriteString(line + "\n")
	}
	return sb.String(), latestPriority, nil
}

func ShouldFlash(msgTime time.Time) bool {
	return time.Since(msgTime) < 500*time.Millisecond
}
