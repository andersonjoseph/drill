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
	width    int
	height   int
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
	switch msg := msg.(type) {
	case messages.UpdateContent:
		m.updateContent()
		return m, nil

	case tea.WindowSizeMsg:
		m.handleResize(msg.Height, msg.Width)
		return m, nil
	}

	return m, nil
}

func (m Model) View() string {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		Height(m.height).
		Width(m.width).
		Render(m.content)
}

func (m *Model) updateContent() {
	var err error
	m.content, err = m.debugger.GetCurrentFileContent((m.height / 2) - 2)
	if err != nil {
		m.Error = fmt.Errorf("error updating content: %w", err)
	}
}

func (m *Model) handleResize(h, w int) {
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
