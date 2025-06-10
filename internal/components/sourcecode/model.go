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
	case messages.WindowFocused:
		m.IsFocused = int(msg) == m.ID

		m.viewport.setFocus(m.IsFocused)
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd

	case messages.RefreshContent:
		if err := m.updateContent(); err != nil {
			return m, func() tea.Msg { return messages.Error(err) }
		}

		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(messages.RefreshContent{})
		return m, cmd

	case messages.DebuggerRestarted, messages.DebuggerStepped:
		if err := m.updateContent(); err != nil {
			return m, func() tea.Msg {
				return messages.Error(fmt.Errorf("error handling debugger step: %w", err))
			}
		}

		line, err := m.debugger.CurrentLine()
		if err != nil {
			return m, func() tea.Msg {
				return messages.Error(fmt.Errorf("error updating content: %w", err))
			}
		}
		m.viewport.jumpToLine(line)

		return m, nil

	case messages.DebuggerBreakpointCreated, messages.DebuggerBreakpointToggled:
		if err := m.updateContent(); err != nil {
			return m, func() tea.Msg {
				return messages.Error(fmt.Errorf("error handling debugger step: %w", err))
			}
		}

		return m, nil

	case tea.KeyMsg:
		if !m.IsFocused {
			return m, nil
		}

		if msg.String() == "n" {
			if err := m.next(); err != nil {
				return m, func() tea.Msg { return messages.Error(err) }
			}
			return m, func() tea.Msg { return messages.DebuggerStepped{} }
		}

		if msg.String() == "c" {
			m.debugger.Continue()
			return m, func() tea.Msg { return messages.DebuggerStepped{} }
		}

		if msg.String() == "r" {
			if err := m.debugger.Restart(); err != nil {
				return m, func() tea.Msg {
					return messages.Error(fmt.Errorf("error restarting: %w", err))
				}
			}
			return m, func() tea.Msg { return messages.DebuggerRestarted{} }
		}

		if msg.String() == "b" {
			if err := m.createBreakpoint(); err != nil {
				return m, func() tea.Msg {
					return messages.Error(err)
				}
			}

			return m, func() tea.Msg {
				return messages.DebuggerBreakpointCreated{}
			}
		}

		if msg.String() == "s" {
			if err := m.debugger.StepIn(); err != nil {
				return m, func() tea.Msg {
					return messages.Error(err)
				}
			}

			return m, func() tea.Msg {
				return messages.DebuggerStepped{}
			}
		}

		if msg.String() == "S" {
			if err := m.debugger.StepOut(); err != nil {
				return m, func() tea.Msg {
					return messages.Error(err)
				}
			}

			return m, func() tea.Msg {
				return messages.DebuggerStepped{}
			}
		}

		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)

		return m, cmd
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
	content, err := m.debugger.CurrentFileContent()
	if err != nil {
		return fmt.Errorf("error updating content: %w", err)
	}

	m.viewport.setContent(content)

	m.currentFilename, err = m.debugger.CurrentFilename()
	if err != nil {
		return fmt.Errorf("error getting current file: %w", err)
	}

	return nil
}

func (m *Model) next() error {
	err := m.debugger.Next()
	if err != nil {
		return messages.Error(fmt.Errorf("error stepping over: %w", err))
	}

	line, err := m.debugger.CurrentLine()
	if err != nil {
		return messages.Error(fmt.Errorf("error stepping over: %w", err))
	}

	m.viewport.jumpToLine(line)
	return nil
}

func (m Model) createBreakpoint() error {
	currentLine := m.viewport.CurrentLineNumber()
	if _, err := m.debugger.CreateBreakpoint(m.currentFilename, currentLine); err != nil {

		if strings.Contains(err.Error(), "Breakpoint exists") {
			filename, err := m.debugger.CurrentFilename()
			if err != nil {
				return messages.Error(fmt.Errorf("error creating breakpoint: getCurrentFilename %w", err))
			}
			bps, err := m.debugger.FileBreakpoints(filename)
			if err != nil {
				return messages.Error(fmt.Errorf("error creating breakpoint: getFileBreakpoints %w", err))
			}

			m.debugger.ClearBreakpoint(bps[currentLine].ID)

			return nil
		}

		return messages.Error(fmt.Errorf("error creating breakpoint: %w", err))
	}

	return nil
}
