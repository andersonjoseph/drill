package alert

import (
	"github.com/andersonjoseph/drill/internal/components"
	"github.com/andersonjoseph/drill/internal/messages"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
)

var (
	alertStyle = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(components.ColorOrange).Foreground(components.ColorOrange)
)

type Model struct {
	message   string
	IsVisible bool
	width     int
	height    int
}

func New(message string) Model {
	return Model{message: message}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		m.IsVisible = false
		return m, nil

	case messages.Error:
		m.message = msg.Error()
		m.IsVisible = true
		return m, nil

	}

	return m, nil
}

func (m Model) View() string {
	if !m.IsVisible {
		return ""
	}
	return alertStyle.Width(m.width).Height(m.height).Render(wordwrap.String(m.message, m.width))
}
