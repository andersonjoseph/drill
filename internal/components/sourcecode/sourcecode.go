package sourcecode

import (
	"github.com/andersonjoseph/drill/internal/messages"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	content string
	width   int
	height  int
}

func New(content string) model {
	return model{
		content: content,
	}
}

func (m model) Init() tea.Cmd { return nil }
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		sidebarWidth := msg.Width/3
		if sidebarWidth >= 40 {
			sidebarWidth = 40
		} else if sidebarWidth <= 20 {
			sidebarWidth = 20
		}
		m.width = (msg.Width - sidebarWidth)-4

		m.height = msg.Height - 3
		return m, nil

	case messages.NewCodeContent:
		m.content = string(msg)
		return m, nil
	}

	return m, nil
}

func (m model) View() string {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		Height(m.height).
		Width(m.width).
		Render(m.content)
}
