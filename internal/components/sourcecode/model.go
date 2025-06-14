package sourcecode

import (
	"fmt"

	"github.com/andersonjoseph/drill/internal/debugger"
	"github.com/andersonjoseph/drill/internal/messages"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	ID        int
	title     string
	IsFocused bool
	width     int
	height    int
	viewport  viewportWithCursorModel
	debugger  *debugger.Debugger
}

func New(id int, title string, d *debugger.Debugger) Model {
	return Model{
		ID:       id,
		title:    title,
		debugger: d,
		viewport: newViewportWithCursor(d),
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case messages.FileRequested:
		m.viewport, cmd = m.viewport.Update(msg)
		return m, tea.Batch(
			cmd,
			func() tea.Msg {
				return messages.WindowFocused(m.ID)
			},
		)

	case messages.WindowFocused:
		m.IsFocused = int(msg) == m.ID
		return m, cmd

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if !m.IsFocused {
		return m, nil
	}
	if msg.String() == "n" {
		if err := m.next(); err != nil {
			return m, messages.ErrorCmd(err)
		}
		return m, func() tea.Msg { return messages.DebuggerStepped{} }
	}

	if msg.String() == "c" {
		m.debugger.Continue()
		return m, func() tea.Msg { return messages.DebuggerStepped{} }
	}

	if msg.String() == "r" {
		if err := m.debugger.Restart(); err != nil {
			return m, messages.ErrorCmd(fmt.Errorf("error restarting: %w", err))
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
			return m, messages.ErrorCmd(err)
		}

		return m, func() tea.Msg {
			return messages.DebuggerStepped{}
		}
	}

	if msg.String() == "S" {
		if err := m.debugger.StepOut(); err != nil {
			return m, messages.ErrorCmd(err)
		}

		return m, func() tea.Msg {
			return messages.DebuggerStepped{}
		}
	}

	if msg.String() == "enter" {
		id, err := m.selectBreakpoint()
		if err != nil {
			return m, messages.ErrorCmd(err)
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

func (m Model) View() string { return m.viewport.View() }

func (m *Model) next() error {
	err := m.debugger.Next()
	if err != nil {
		return fmt.Errorf("error stepping over: %w", err)
	}

	return nil
}

func (m Model) createOrToggleBreakpoint() tea.Cmd {
	bp, ok, err := m.currentBreakpoint()
	if err != nil {
		return messages.ErrorCmd(fmt.Errorf("error toggling breakpoint: currentBreakpoint %w", err))
	}

	if !ok {
		currentLine := m.viewport.CurrentLineNumber()
		bp, err := m.debugger.CreateBreakpoint(m.viewport.filename, currentLine)
		if err != nil {
			return func() tea.Msg {
				return messages.Error(fmt.Errorf("error creating breakpoint: %w", err))
			}
		}

		return messages.DebuggerBreakpointCreatedCmd(bp.ID, bp.Filename, currentLine)
	}

	m.debugger.ToggleBreakpoint(bp.ID)
	return messages.DebuggerBreakpointToggledCmd(bp.ID, bp.Filename, bp.Line)
}

func (m Model) clearBreakpoint() tea.Cmd {
	bp, ok, err := m.currentBreakpoint()
	if err != nil {
		return messages.ErrorCmd(fmt.Errorf("error clearing breakpoint: currentBreakpoint %w", err))
	}
	if !ok {
		return nil
	}

	if err := m.debugger.ClearBreakpoint(bp.ID); err != nil {
		return messages.ErrorCmd(fmt.Errorf("error clearing breakpoint %w", err))
	}

	return messages.DebuggerBreakpointClearedCmd(bp.ID, bp.Filename, bp.Line)
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

	bps, err := m.debugger.FileBreakpoints(m.viewport.filename)
	if err != nil {
		return debugger.Breakpoint{}, false, fmt.Errorf("error toggling breakpoint: currentFilename %w", err)
	}

	bp, ok := bps[currentLine]
	if !ok {
		return debugger.Breakpoint{}, false, nil
	}

	return bp, true, nil
}
