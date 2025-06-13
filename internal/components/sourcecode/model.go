package sourcecode

import (
	"fmt"

	"github.com/andersonjoseph/drill/internal/debugger"
	"github.com/andersonjoseph/drill/internal/messages"
	tea "github.com/charmbracelet/bubbletea"
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

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
				return messages.Error(fmt.Errorf("error handling debugger step: %w", err))
			}
		}
		m.viewport.jumpToLine(line)

		return m, nil

	case messages.DebuggerBreakpointCreated, messages.DebuggerBreakpointToggled, messages.DebuggerBreakpointCleared:
		if err := m.updateContent(); err != nil {
			return m, func() tea.Msg {
				return messages.Error(fmt.Errorf("error handling debugger step: %w", err))
			}
		}

		return m, nil

	case messages.OpenedFile:
		f, err := m.debugger.GoToFile(msg.Filename)
		if err != nil {
			return m, func() tea.Msg {
				return messages.Error(fmt.Errorf("error handling OpenedFile: %w", err))
			}
		}

		m.currentFilename = f.Filename
		m.viewport.setContent(f.Content)
		m.viewport.jumpToLine(msg.Line)

		return m, func() tea.Msg {
			return messages.WindowFocused(m.ID)
		}

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
			return m, m.createOrToggleBreakpoint()
		}

		if msg.String() == "d" {
			return m, m.clearBreakpoint()
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

		if msg.String() == "z" {
			m.viewport.centerCursorView()

			return m, nil
		}

		if msg.String() == "enter" {
			id, err := m.selectBreakpoint()
			if err != nil {
				return m, func() tea.Msg {
					return messages.Error(err)
				}
			}
			if id == 0 {
				return m, nil
			}

			return m, func() tea.Msg {
				return messages.BreakpointSelected(id)
			}
		}

		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)

		return m, cmd
	}

	return m, nil
}

func (m Model) View() string {
	return m.viewport.View()
}

func (m *Model) updateContent() error {
	f, err := m.debugger.GoToCurrentFile()
	if err != nil {
		return fmt.Errorf("error updating content: %w", err)
	}

	m.currentFilename = f.Filename
	m.viewport.setContent(f.Content)

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

func (m Model) createOrToggleBreakpoint() tea.Cmd {
	bp, ok, err := m.currentBreakpoint()
	if err != nil {
		return func() tea.Msg {
			return messages.Error(fmt.Errorf("error clearing breakpoint: currentBreakpoint %w", err))
		}
	}

	if !ok {
		currentLine := m.viewport.CurrentLineNumber()
		if _, err := m.debugger.CreateBreakpoint(m.currentFilename, currentLine); err != nil {
			return func() tea.Msg {
				return messages.Error(fmt.Errorf("error creating breakpoint: %w", err))
			}
		}

		return func() tea.Msg {
			return messages.DebuggerBreakpointCreated{}
		}
	}

	m.debugger.ToggleBreakpoint(bp.ID)
	return func() tea.Msg {
		return messages.DebuggerBreakpointToggled{}
	}
}

func (m Model) clearBreakpoint() tea.Cmd {
	bp, ok, err := m.currentBreakpoint()
	if err != nil {
		return func() tea.Msg {
			return messages.Error(fmt.Errorf("error clearing breakpoint: currentBreakpoint %w", err))
		}
	}
	if !ok {
		return nil
	}

	if err := m.debugger.ClearBreakpoint(bp.ID); err != nil {
		return func() tea.Msg {
			return messages.Error(fmt.Errorf("error clearing breakpoint %w", err))
		}
	}
	return func() tea.Msg {
		return messages.DebuggerBreakpointCleared{}
	}
}

func (m Model) selectBreakpoint() (int, error) {
	bp, ok, err := m.currentBreakpoint()
	if err != nil {
		return 0, fmt.Errorf("error selecting breakpoint: %w", err)
	}
	if !ok {
		return 0, nil
	}

	return bp.ID, nil
}

func (m Model) currentBreakpoint() (debugger.Breakpoint, bool, error) {
	currentLine := m.viewport.CurrentLineNumber()

	f, err := m.debugger.GoToCurrentFile()
	if err != nil {
		return debugger.Breakpoint{}, false, messages.Error(fmt.Errorf("error toggling breakpoint: currentFilename %w", err))
	}

	bps, err := m.debugger.FileBreakpoints(f.Filename)
	if err != nil {
		return debugger.Breakpoint{}, false, messages.Error(fmt.Errorf("error toggling breakpoint: currentFilename %w", err))
	}

	bp, ok := bps[currentLine]
	if !ok {
		return debugger.Breakpoint{}, false, nil
	}

	return bp, true, nil
}
