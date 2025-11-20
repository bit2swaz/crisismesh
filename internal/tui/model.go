package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bit2swaz/crisismesh/internal/core"
	"github.com/bit2swaz/crisismesh/internal/store"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"gorm.io/gorm"
)

type tickMsg time.Time

var (
	activePeerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
	inactivePeerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // Faded gray
	peerListStyle     = lipgloss.NewStyle().Width(25).Border(lipgloss.NormalBorder(), false, true, false, false).PaddingRight(1)
	viewportStyle     = lipgloss.NewStyle().PaddingLeft(1)
)

type model struct {
	db          *gorm.DB
	nodeID      string
	peers       []store.Peer
	viewport    viewport.Model
	textInput   textinput.Model
	chatHistory string
	ready       bool
	err         error
}

func initialModel(db *gorm.DB, nodeID string) model {
	ti := textinput.New()
	ti.Placeholder = "Type a message..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	// Initial peer load
	var peers []store.Peer
	db.Find(&peers)
	sortPeers(peers)

	history, _ := buildChatHistory(db, nodeID)
	if history == "" {
		history = "Welcome to CrisisMesh!\nChat history will appear here.\n"
	}

	return model{
		db:          db,
		nodeID:      nodeID,
		peers:       peers,
		textInput:   ti,
		chatHistory: history,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, tick())
}

func tick() tea.Cmd {
	return tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	switch msg := msg.(type) {
	case tickMsg:
		var peers []store.Peer
		m.db.Find(&peers)
		sortPeers(peers)
		m.peers = peers

		// Refresh chat history
		newHistory, err := buildChatHistory(m.db, m.nodeID)
		if err == nil && newHistory != m.chatHistory {
			m.chatHistory = newHistory
			m.viewport.SetContent(m.chatHistory)
			// Auto-scroll to bottom on new messages
			// In a real app, we'd check if user is scrolled up
			m.viewport.GotoBottom()
		}

		return m, tick()

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			// Mock sending message
			if m.textInput.Value() != "" {
				txt := m.textInput.Value()
				ts := time.Now().Unix()

				// Save to DB
				msgID := core.GenerateMessageID(m.nodeID, txt, ts)
				msg := &store.Message{
					ID:        msgID,
					SenderID:  m.nodeID,
					Content:   txt,
					Timestamp: ts,
					Status:    "sent",
				}
				store.SaveMessage(m.db, msg)

				content := fmt.Sprintf("You: %s\n", txt)
				m.chatHistory += content
				m.viewport.SetContent(m.chatHistory)
				m.viewport.GotoBottom()
				m.textInput.Reset()
			}
		}

	case tea.WindowSizeMsg:
		headerHeight := 0
		footerHeight := 1 // Input line
		verticalMarginHeight := headerHeight + footerHeight

		if !m.ready {
			// Since this is called when the program starts, we can initialize the viewport here
			m.viewport = viewport.New(msg.Width-peerListStyle.GetWidth(), msg.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.viewport.SetContent(m.chatHistory)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width - peerListStyle.GetWidth()
			m.viewport.Height = msg.Height - verticalMarginHeight
		}
	}

	m.textInput, tiCmd = m.textInput.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	peerList := m.renderPeerList()
	chatView := viewportStyle.Render(m.viewport.View())

	// Join peer list and chat view horizontally
	mainArea := lipgloss.JoinHorizontal(lipgloss.Top, peerList, chatView)

	// Join main area and input vertically
	return lipgloss.JoinVertical(lipgloss.Left, mainArea, m.textInput.View())
}

func (m model) renderPeerList() string {
	var s strings.Builder
	s.WriteString("PEERS\n-----\n")
	for _, p := range m.peers {
		nick := p.Nick
		if nick == "" {
			nick = "Unknown"
		}

		if p.IsActive {
			s.WriteString(activePeerStyle.Render(nick) + "\n")
		} else {
			s.WriteString(inactivePeerStyle.Render(nick) + "\n")
		}
	}
	return peerListStyle.Render(s.String())
}

func sortPeers(peers []store.Peer) {
	sort.Slice(peers, func(i, j int) bool {
		// Active peers first
		if peers[i].IsActive && !peers[j].IsActive {
			return true
		}
		if !peers[i].IsActive && peers[j].IsActive {
			return false
		}
		// Then by Nickname
		return peers[i].Nick < peers[j].Nick
	})
}

func buildChatHistory(db *gorm.DB, nodeID string) (string, error) {
	var sb strings.Builder
	sb.WriteString("Welcome to CrisisMesh!\nChat history will appear here.\n")

	// Load existing messages
	msgs, err := store.GetMessages(db, 50)
	if err != nil {
		return "", err
	}
	// Reverse messages to show oldest first
	for i := len(msgs) - 1; i >= 0; i-- {
		msg := msgs[i]
		sender := "Peer"
		if msg.SenderID == nodeID {
			sender = "You"
		}
		sb.WriteString(fmt.Sprintf("%s: %s\n", sender, msg.Content))
	}
	return sb.String(), nil
}

// StartTUI initializes and runs the TUI program
func StartTUI(db *gorm.DB, nodeID string) error {
	p := tea.NewProgram(initialModel(db, nodeID), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
