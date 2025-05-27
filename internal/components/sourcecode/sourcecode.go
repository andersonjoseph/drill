package sourcecode

import (
	"github.com/andersonjoseph/drill/internal/messages"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	content string
}

func New(content string) model {
	return model{
		content: content,
	}
}

func (m model) Init() tea.Cmd { return nil }
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.NewCodeContent:
		m.content = string(msg)
		return m, nil
	}

	return m, nil
}

func (m model) View() string {
	return lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Width(150).Render(m.content)
}
