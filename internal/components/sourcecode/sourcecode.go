package sourcecode

import (
	"github.com/andersonjoseph/drill/internal/debugger"
	"github.com/andersonjoseph/drill/internal/messages"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	content  string
	width    int
	height   int
	debugger *debugger.Debugger
}

func New(d *debugger.Debugger) model {
	return model{
		debugger: d,
	}
}

func (m model) Init() tea.Cmd { return nil }
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.UpdateContent:
		m.updateContent()

	case tea.WindowSizeMsg:
		m.handleResize(msg.Height, msg.Width)
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

func (m *model) updateContent() {
	m.content = m.debugger.GetCurrentFileContent((m.height / 2) - 2)
}

func (m *model) handleResize(h, w int) {
	sidebarWidth := w / 3
	if sidebarWidth >= 40 {
		sidebarWidth = 40
	} else if sidebarWidth <= 20 {
		sidebarWidth = 20
	}
	m.width = (w - sidebarWidth) - 4
	m.height = max(h-2, 5)

	m.updateContent()
}
