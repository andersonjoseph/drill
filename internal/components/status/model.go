package status

import (
	"strings"

	"github.com/andersonjoseph/drill/internal/components"
	"github.com/andersonjoseph/drill/internal/messages"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	alertStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder(), true).
			BorderForeground(components.ColorOrange).
			Foreground(components.ColorOrange)

	hintStyle = lipgloss.NewStyle().
			Foreground(components.ColorPurple)
)

type Model struct {
	width   int
	height  int
	content string
	error   error
}

func New() Model {
	return Model{}
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

	case messages.Error:
		m.error = msg
		return m, nil

	case messages.UpdatedHint:
		m.content = string(msg)
		return m, nil

	case tea.KeyMsg:
		m.error = nil
		return m, nil
	}

	return m, nil
}

func (m Model) View() string {
	if m.error != nil {
		msg := m.error.Error()
		if strings.Contains(m.error.Error(), "has exited with status 0") {
			msg = "debugger session ended, press r to restart or q to quit"
		}

		return alertStyle.Width(m.width).Render(msg)
	}

	return hintStyle.Width(m.width).Render("1-5: navigate,", m.content)
}
