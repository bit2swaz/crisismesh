package tui

import (
	"sort"
	"strings"
	"time"

	"github.com/bit2swaz/crisismesh/internal/store"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
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
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		tick(),
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
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
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
