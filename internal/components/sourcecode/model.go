package sourcecode

import (
	"fmt"

	"github.com/andersonjoseph/drill/internal/debugger"
	"github.com/andersonjoseph/drill/internal/messages"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	content  string
	Width    int
	Height   int
	Error    error
	debugger *debugger.Debugger
}

func New(d *debugger.Debugger) Model {
	return Model{
		debugger: d,
	}
}

func (m Model) Init() tea.Cmd { return nil }
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg.(type) {
	case messages.UpdateContent, tea.WindowSizeMsg:
		m.updateContent()
		return m, nil
	}

	return m, nil
}

func (m Model) View() string {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		Height(m.Height).
		Width(m.Width).
		Render(m.content)
}

func (m *Model) updateContent() {
	var err error
	m.content, err = m.debugger.GetCurrentFileContent((m.Height / 2) - 2)
	if err != nil {
		m.Error = fmt.Errorf("error updating content: %w", err)
	}
}
