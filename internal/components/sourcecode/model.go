package sourcecode

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/alecthomas/chroma/v2/quick"
	"github.com/andersonjoseph/drill/internal/components"
	"github.com/andersonjoseph/drill/internal/debugger"
	"github.com/andersonjoseph/drill/internal/messages"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	listFocusedStyle lipgloss.Style = lipgloss.NewStyle().Foreground(components.ColorGreen)
	listDefaultStyle lipgloss.Style = lipgloss.NewStyle().Foreground(components.ColorWhite)
)

type Model struct {
	id              int
	title           string
	currentFilename string
	isFocused       bool
	content         string
	Width           int
	Height          int
	Error           error
	debugger        *debugger.Debugger
}

func New(id int, title string, d *debugger.Debugger) Model {
	return Model{
		id:       id,
		title:    title,
		debugger: d,
	}
}

func (m Model) Init() tea.Cmd { return nil }
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.UpdateContent, tea.WindowSizeMsg:
		m.updateContent()
		return m, nil

	case tea.KeyMsg:
		if id, err := strconv.Atoi(msg.String()); err == nil {
			m.isFocused = id == m.id
			return m, nil
		}
		if !m.isFocused {
			return m, nil
		}

		if msg.String() == "n" {
			_, err := m.debugger.Client.Next()
			if err != nil {
				m.Error = fmt.Errorf("error getting debugger state: %w", err)
				return m, nil
			}
			return m, func() tea.Msg {
				return messages.UpdateContent{}
			}
		}

		if msg.String() == "c" {
			<-m.debugger.Client.Continue()
			return m, func() tea.Msg {
				return messages.UpdateContent{}
			}
		}
	}

	return m, nil
}

func (m Model) View() string {
	var style lipgloss.Style
	if m.isFocused {
		style = listFocusedStyle
	} else {
		style = listDefaultStyle
	}

	title := fmt.Sprintf("[%d] %s [%s]", m.id, m.title, m.currentFilename)

	topBorder := "┌" + title + strings.Repeat("─", max(m.Width-len(title), 1)) + "┐"

	return lipgloss.JoinVertical(
		lipgloss.Top,
		style.Render(topBorder),
		lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderTop(false).
			BorderForeground(style.GetForeground()).
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
