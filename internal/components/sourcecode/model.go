package sourcecode

import (
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/v2/quick"
	"github.com/andersonjoseph/drill/internal/debugger"
	"github.com/andersonjoseph/drill/internal/messages"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	title           string
	currentFilename string
	content         string
	Width           int
	Height          int
	Error           error
	debugger        *debugger.Debugger
}

func New(title string, d *debugger.Debugger) Model {
	return Model{
		title:    title,
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
	title := fmt.Sprintf("%s [%s]", m.title, m.currentFilename)

	topBorder := "┌" + title + strings.Repeat("─", max(m.Width-len(title), 1)) + "┐"

	return lipgloss.JoinVertical(
		lipgloss.Top,
		topBorder,
		lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderTop(false).
			Height(m.Height).
			Width(m.Width).
			Render(m.content),
	)
}

func colorize(content string) (string, error) {
	sb := strings.Builder{}

	err := quick.Highlight(&sb, content, "go", "terminal8", "native")
	if err != nil {
		return "", fmt.Errorf("error highlighting the source code: %w", err)
	}

	return sb.String(), nil
}

func (m *Model) updateContent() {
	var err error
	content, err := m.debugger.GetCurrentFileContent((m.Height / 2) - 2)
	if err != nil {
		m.Error = fmt.Errorf("error updating content: %w", err)
	}
	m.content, err = colorize(content)
	if err != nil {
		m.Error = fmt.Errorf("error colorizing content: %w", err)
	}

	m.currentFilename, err = m.debugger.GetCurrentFilename()
	if err != nil {
		m.Error = fmt.Errorf("error getting the current file: %w", err)
	}
}
