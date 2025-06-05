package sourcecode

import (
	"fmt"
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
	ID              int
	title           string
	currentFilename string
	IsFocused       bool
	content         string
	Width           int
	Height          int
	Error           error
	debugger        *debugger.Debugger
}

func New(id int, title string, d *debugger.Debugger) Model {
	return Model{
		ID:       id,
		title:    title,
		debugger: d,
	}
}

func (m Model) Init() tea.Cmd { return nil }
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.IsFocused:
		m.IsFocused = bool(msg)
		return m, nil

	case messages.UpdateContent, tea.WindowSizeMsg:
		m.updateContent()
		return m, nil

	case tea.KeyMsg:
		if !m.IsFocused {
			return m, nil
		}

		if msg.String() == "n" {
			_, err := m.debugger.Client.Next()
			if err != nil {
				m.Error = fmt.Errorf("error stepping over to the next line: %w", err)
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

		if msg.String() == "r" {
			_, err := m.debugger.Client.Restart(false)
			if err != nil {
				m.Error = fmt.Errorf("error restarting debugger: %w", err)
				return m, nil
			}
			m.updateContent()
			return m, func() tea.Msg {
				return messages.Restart{}
			}
		}
	}

	return m, nil
}

func (m Model) View() string {
	var style lipgloss.Style
	if m.IsFocused {
		style = listFocusedStyle
	} else {
		style = listDefaultStyle
	}

	title := fmt.Sprintf("[%d] %s [%s]", m.ID, m.title, m.currentFilename)

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
