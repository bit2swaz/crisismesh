package tui

import (
	"sort"
	"strings"
	"time"

	"github.com/bit2swaz/crisismesh/internal/store"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"gorm.io/gorm"
)

type tickMsg time.Time
type flashMsg int

const (
	TabComms = iota
	TabNetwork
	TabGuide
)

type Publisher interface {
	PublishText(content string) error
	ManualConnect(addr string) error
	BroadcastSafe() error
}

type keyMap struct {
	Tab  key.Binding
	Quit key.Binding
	Help key.Binding
	Safe key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Tab, k.Safe},
		{k.Help, k.Quit},
	}
}

var keys = keyMap{
	Tab: key.NewBinding(
		key.WithKeys("f1", "f2", "f3"),
		key.WithHelp("F1-F3", "switch tabs"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c", "esc"),
		key.WithHelp("ctrl+c", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	Safe: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "broadcast safe"),
	),
}

type model struct {
	db          *gorm.DB
	nodeID      string
	peers       []store.Peer
	viewport    viewport.Model
	textInput   textinput.Model
	chatHistory string
	ready       bool
	err         error
	msgSub      <-chan store.Message
	peerSub     <-chan []store.Peer
	publisher   Publisher

	// New fields
	activeTab int
	flashTick int
	startTime time.Time
	spinner   spinner.Model
	help      help.Model
	keys      keyMap
}

func initialModel(db *gorm.DB, nodeID string, msgSub <-chan store.Message, peerSub <-chan []store.Peer, pub Publisher) model {
	ti := textinput.New()
	ti.Placeholder = "Type a message..."
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	var peers []store.Peer
	db.Find(&peers)
	sortPeers(peers)

	history, _ := buildChatHistory(db, nodeID)
	if history == "" {
		history = "Welcome to CrisisMesh!\nChat history will appear here.\n"
	}

	s := spinner.New()
	s.Spinner = spinner.Pulse
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return model{
		db:          db,
		nodeID:      nodeID,
		peers:       peers,
		textInput:   ti,
		chatHistory: history,
		msgSub:      msgSub,
		peerSub:     peerSub,
		publisher:   pub,
		activeTab:   TabComms,
		startTime:   time.Now(),
		spinner:     s,
		help:        help.New(),
		keys:        keys,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		tick(),
		m.spinner.Tick,
		WaitForUpdates(m.msgSub),
		WaitForPeerUpdates(m.peerSub),
	)
}

func WaitForUpdates(sub <-chan store.Message) tea.Cmd {
	return func() tea.Msg {
		return <-sub
	}
}

func WaitForPeerUpdates(sub <-chan []store.Peer) tea.Cmd {
	return func() tea.Msg {
		return <-sub
	}
}

func tick() tea.Cmd {
	return tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func flash() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return flashMsg(1)
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)
	switch msg := msg.(type) {
	case store.Message:
		newHistory, err := buildChatHistory(m.db, m.nodeID)
		if err == nil {
			m.chatHistory = newHistory
			m.viewport.SetContent(m.chatHistory)
			m.viewport.GotoBottom()
		}
		var cmds []tea.Cmd
		cmds = append(cmds, WaitForUpdates(m.msgSub))
		if msg.Priority == 2 {
			m.flashTick = 10 // Flash for 1 second (10 * 100ms)
			cmds = append(cmds, flash())
		}
		return m, tea.Batch(cmds...)
	case flashMsg:
		if m.flashTick > 0 {
			m.flashTick--
			return m, flash()
		}
		return m, nil
	case []store.Peer:
		m.peers = msg
		sortPeers(m.peers)
		return m, WaitForPeerUpdates(m.peerSub)
	case tickMsg:
		var peers []store.Peer
		m.db.Find(&peers)
		sortPeers(peers)
		m.peers = peers
		newHistory, err := buildChatHistory(m.db, m.nodeID)
		if err == nil && newHistory != m.chatHistory {
			m.chatHistory = newHistory
			m.viewport.SetContent(m.chatHistory)
			m.viewport.GotoBottom()
		}
		return m, tick()
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
		case key.Matches(msg, m.keys.Tab):
			// Cycle tabs for simplicity or keep F-keys if preferred, but prompt said "Bind ? key".
			// The existing code used F1, F2, F3. I'll keep F-keys logic but maybe map them in keyMap.
			// Actually, I defined Tab as F1-F3 in keyMap.
			// But key.Matches checks if the key matches the binding.
			// If I want to distinguish F1, F2, F3, I should probably check msg.String() or have separate bindings.
			// For now, let's stick to the existing logic for F-keys but add Help toggle.
		}

		switch msg.Type {
		case tea.KeyF1:
			m.activeTab = TabComms
		case tea.KeyF2:
			m.activeTab = TabNetwork
		case tea.KeyF3:
			m.activeTab = TabGuide
		case tea.KeyCtrlS:
			if err := m.publisher.BroadcastSafe(); err != nil {
			}
		case tea.KeyEnter:
			if m.textInput.Value() != "" {
				txt := m.textInput.Value()
				if strings.HasPrefix(txt, "/connect ") {
					addr := strings.TrimPrefix(txt, "/connect ")
					if err := m.publisher.ManualConnect(addr); err != nil {
					}
				} else if txt == "/safe" {
					if err := m.publisher.BroadcastSafe(); err != nil {
					}
				} else {
					if err := m.publisher.PublishText(txt); err != nil {
					}
				}
				m.textInput.Reset()
			}
		}
	case tea.WindowSizeMsg:
		headerHeight := 8 // Increased for HUD
		footerHeight := 3 // Increased for Status Bar
		verticalMarginHeight := headerHeight + footerHeight
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.viewport.SetContent(m.chatHistory)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}
	}
	m.textInput, tiCmd = m.textInput.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)
	return m, tea.Batch(tiCmd, vpCmd)
}

func sortPeers(peers []store.Peer) {
	sort.Slice(peers, func(i, j int) bool {
		if peers[i].IsActive && !peers[j].IsActive {
			return true
		}
		if !peers[i].IsActive && peers[j].IsActive {
			return false
		}
		return peers[i].Nick < peers[j].Nick
	})
}

func StartTUI(db *gorm.DB, nodeID string, msgSub <-chan store.Message, peerSub <-chan []store.Peer, pub Publisher) error {
	p := tea.NewProgram(initialModel(db, nodeID, msgSub, peerSub, pub), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
