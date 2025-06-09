package sourcecode

import (
	"fmt"
	"strings"

	"github.com/andersonjoseph/drill/internal/components"
	"github.com/andersonjoseph/drill/internal/debugger"
	"github.com/andersonjoseph/drill/internal/messages"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	windowFocusedStyle = lipgloss.NewStyle().
				Foreground(components.ColorGreen).
				Border(lipgloss.NormalBorder()).
				BorderTop(false).
				BorderForeground(components.ColorGreen)

	windowDefaultStyle = lipgloss.NewStyle().
				Foreground(components.ColorWhite).
				Border(lipgloss.NormalBorder()).
				BorderTop(false).
				BorderForeground(components.ColorWhite)
)

type Model struct {
	ID              int
	title           string
	currentFilename string
	IsFocused       bool
	cursor          int
	width           int
	height          int
	viewport        viewportWithCursorModel
	debugger        *debugger.Debugger
}

func New(id int, title string, d *debugger.Debugger) Model {
	return Model{
		ID:       id,
		title:    title,
		debugger: d,
		viewport: newViewportWithCursor(),
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.IsFocused:
		var cmd tea.Cmd

		m.IsFocused = bool(msg)
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd

	case messages.UpdateContent:
		if err := m.updateContent(); err != nil {
			return m, func() tea.Msg { return messages.Error(err) }
		}

		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)

		return m, cmd

	case tea.KeyMsg:
		if !m.IsFocused {
			return m, nil
		}

		switch msg.String() {

		case "n":
			debuggerState, err := m.debugger.Client.Next()
			if err != nil {
				return m, func() tea.Msg {
					return messages.Error(fmt.Errorf("error stepping over: %w", err))
				}
			}

			m.viewport.jumpToLine(debuggerState.CurrentThread.Line)
			return m, func() tea.Msg { return messages.UpdateContent{} }

		case "c":
			debuggerState := <-m.debugger.Client.Continue()
			m.viewport.jumpToLine(debuggerState.CurrentThread.Line)

			return m, func() tea.Msg { return messages.UpdateContent{} }

		case "r":
			if _, err := m.debugger.Client.Restart(false); err != nil {
				return m, func() tea.Msg {
					return messages.Error(fmt.Errorf("error restarting: %w", err))
				}
			}
			if err := m.updateContent(); err != nil {
				return m, func() tea.Msg { return messages.Error(err) }
			}
			return m, func() tea.Msg { return messages.Restart{} }

		default:
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m Model) View() string {
	style := windowDefaultStyle
	if m.IsFocused {
		style = windowFocusedStyle
	}

	title := fmt.Sprintf("[%d] %s [%s]", m.ID, m.title, m.currentFilename)
	topBorder := "┌" + title + strings.Repeat("─", max(m.width-len(title), 1)) + "┐"

	return lipgloss.JoinVertical(
		lipgloss.Top,
		style.Border(lipgloss.Border{}).Render(topBorder),
		style.Height(m.height).Width(m.width).Render(m.viewport.View()),
	)
}

func (m *Model) updateContent() error {
	content, err := m.debugger.GetCurrentFileContent()
	if err != nil {
		return fmt.Errorf("error updating content: %w", err)
	}

	m.viewport.setContent(strings.Split(content, "\n"))

	m.currentFilename, err = m.debugger.GetCurrentFilename()
	if err != nil {
		return fmt.Errorf("error getting current file: %w", err)
	}

	return nil
}
