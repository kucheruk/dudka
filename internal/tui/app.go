package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

const tickInterval = 500 * time.Millisecond

type tickMsg time.Time

type snapMsg struct {
	snap Snapshot
	err  error
}

type statusErrMsg string

// Model is the interactive bubbletea TUI (P046).
type Model struct {
	client     *Client
	input      textinput.Model
	snap       Snapshot
	statusMsg  string
	feedScroll int
	width      int
	height     int
	quitting   bool
}

// NewModel builds an interactive TUI model bound to engine client.
func NewModel(client *Client) Model {
	ti := textinput.New()
	ti.Placeholder = "текст…  Enter = отправить"
	ti.Prompt = ""
	ti.CharLimit = 4000
	ti.Focus()
	return Model{
		client: client,
		input:  ti,
		width:  80,
		height: 24,
		snap:   Snapshot{EngineOK: false, Err: "загрузка…"},
	}
}

// Init starts refresh + tick.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.refreshCmd(), tickCmd())
}

func tickCmd() tea.Cmd {
	return tea.Tick(tickInterval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m Model) refreshCmd() tea.Cmd {
	c := m.client
	return func() tea.Msg {
		snap, err := c.Fetch()
		if err != nil {
			return snapMsg{snap: Snapshot{EngineOK: false, Err: err.Error()}, err: err}
		}
		return snapMsg{snap: snap}
	}
}

// Update handles keys, resize, ticks, and engine snapshots.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.Width = maxInt(10, m.width-16)
		return m, nil
	case tickMsg:
		return m, tea.Batch(m.refreshCmd(), tickCmd())
	case snapMsg:
		m.snap = msg.snap
		return m, nil
	case statusErrMsg:
		m.statusMsg = string(msg)
		return m, nil
	case scanDoneMsg:
		m.snap = msg.snap
		m.statusMsg = fmt.Sprintf("ПОИСК ЗАВЕРШЁН · найдено: %d", msg.n)
		return m, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			m.quitting = true
			return m, tea.Quit
		case tea.KeyEsc:
			if m.input.Value() != "" {
				m.input.SetValue("")
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit
		case tea.KeyEnter:
			return m.submitCompose()
		case tea.KeyUp:
			m.feedScroll++
			return m, nil
		case tea.KeyDown:
			if m.feedScroll > 0 {
				m.feedScroll--
			}
			return m, nil
		case tea.KeyPgUp:
			m.feedScroll += 5
			return m, nil
		case tea.KeyPgDown:
			m.feedScroll -= 5
			if m.feedScroll < 0 {
				m.feedScroll = 0
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) submitCompose() (tea.Model, tea.Cmd) {
	line := m.input.Value()
	m.input.SetValue("")
	m.statusMsg = ""
	if strings.TrimSpace(line) == "" {
		return m, nil
	}
	if ParseSearchCommand(line) {
		m.statusMsg = "ИЩУ СОСЕДЕЙ…"
		return m, m.scanCmd()
	}
	c := m.client
	return m, func() tea.Msg {
		if err := HandleComposeLine(c, line); err != nil {
			if _, warning := err.(*ErrLargeFileWarning); warning {
				return statusErrMsg(err.Error())
			}
			logTUIError("compose", err)
			return statusErrMsg(composeErrorMessage(line))
		}
		snap, err := c.Fetch()
		if err != nil {
			return snapMsg{snap: Snapshot{EngineOK: false, Err: err.Error()}, err: err}
		}
		return snapMsg{snap: snap}
	}
}

func composeErrorMessage(line string) string {
	switch {
	case strings.HasPrefix(strings.TrimSpace(line), "/nick"):
		return "ИМЯ НЕ ИЗМЕНЕНО · используйте /nick Имя"
	case strings.HasPrefix(strings.TrimSpace(line), "/announce"):
		return "ФАЙЛ НЕ ДОБАВЛЕН · проверьте путь"
	case strings.HasPrefix(strings.TrimSpace(line), "/fetch"):
		return "СКАЧИВАНИЕ НЕ НАЧАТО · проверьте file_id"
	case strings.HasPrefix(strings.TrimSpace(line), "/cancel"):
		return "СКАЧИВАНИЕ НЕ ОСТАНОВЛЕНО · проверьте file_id"
	default:
		return "СООБЩЕНИЕ НЕ ОТПРАВЛЕНО · повторите"
	}
}

func (m Model) scanCmd() tea.Cmd {
	c := m.client
	return func() tea.Msg {
		n, err := c.Scan()
		if err != nil {
			logTUIError("search", err)
			return statusErrMsg("ПОИСК НЕ ЗАВЕРШЁН · повторите /search · подробности в tui.log")
		}
		snap, ferr := c.Fetch()
		if ferr != nil {
			logTUIError("search refresh", ferr)
			return statusErrMsg(fmt.Sprintf("НАЙДЕНО: %d · лента обновится автоматически", n))
		}
		// Prefer snap update; status shown via a tiny wrapper message type.
		return scanDoneMsg{n: n, snap: snap}
	}
}

type scanDoneMsg struct {
	n    int
	snap Snapshot
}

// View renders fixed panels on a full charcoal canvas.
func (m Model) View() string {
	if m.quitting {
		return ""
	}
	content := RenderScreen(ScreenState{
		Snap:       m.snap,
		Compose:    m.input.Value(),
		StatusMsg:  m.statusMsg,
		FeedScroll: m.feedScroll,
		CursorOn:   true,
	}, m.width, m.height)
	return styleCanvas(m.width, m.height).Render(content)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// RunInteractive starts the alt-screen TUI against engineURL (blocks until quit).
func RunInteractive(engineURL string) error {
	// Help Terminal.app / weak profiles pick up DESIGN truecolor tokens.
	if os.Getenv("COLORTERM") == "" {
		_ = os.Setenv("COLORTERM", "truecolor")
	}
	if os.Getenv("TERM") == "dumb" {
		_ = os.Setenv("TERM", "xterm-256color")
	}
	client := NewClient(engineURL)
	p := tea.NewProgram(NewModel(client), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
