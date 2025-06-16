package output

import (
	"github.com/andersonjoseph/drill/internal/components"
	"github.com/andersonjoseph/drill/internal/debugger"
	"github.com/andersonjoseph/drill/internal/messages"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	hintString = "i: command mode, j: down, k: up"
)

var (
	outputContentStyle lipgloss.Style = lipgloss.NewStyle().Foreground(components.ColorWhite)

	stdoutLabelStyle  lipgloss.Style = lipgloss.NewStyle().Foreground(components.ColorGrey)
	stderrLabelStyle  lipgloss.Style = lipgloss.NewStyle().Foreground(components.ColorOrange)
	commandLabelStyle lipgloss.Style = lipgloss.NewStyle().Foreground(components.ColorYellow)
)

type Model struct {
	ID           int
	IsFocused    bool
	content      string
	width        int
	height       int
	viewport     viewport.Model
	commandInput CommandInputModel
	debugger     *debugger.Debugger
}

func New(id int, title string, d *debugger.Debugger) Model {
	m := Model{
		ID:           id,
		debugger:     d,
		content:      "",
		viewport:     viewport.New(30, 5),
		commandInput: newCommandInputModel(id, d),
	}

	return m
}

func waitForDebuggerOutput(c chan debugger.Output) tea.Cmd {
	return func() tea.Msg {
		o := <-c
		switch o.Source {
		case debugger.SourceStderr:
			return messages.DebuggerStderrReceived(o.Content)
		case debugger.SourceCommand:
			return messages.DebuggerCommandReceived(o.Content)
		case debugger.SourceStdout:
			return messages.DebuggerStdoutReceived(o.Content)
		default:
			return nil
		}
	}
}

func (m Model) Init() tea.Cmd {
	return waitForDebuggerOutput(m.debugger.Output)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.WindowFocused:
		m.IsFocused = int(msg) == m.ID
		if !m.IsFocused {
			return m, nil
		}

		return m, func() tea.Msg {
			return messages.UpdatedHint(hintString)
		}

	case messages.DebuggerRestarted:
		m.content = ""
		m.viewport.SetContent(m.content)
		return m, nil

	case messages.DebuggerStdoutReceived:
		label := stdoutLabelStyle.Render("[stdout] ")
		m.content += "\n" + label + string(msg)
		m.viewport.SetContent(m.content)
		m.viewport.GotoBottom()

		return m, waitForDebuggerOutput(m.debugger.Output)

	case messages.DebuggerStderrReceived:
		label := stderrLabelStyle.Render("[stderr] ")
		m.content += "\n" + label + string(msg)
		m.viewport.SetContent(m.content)
		m.viewport.GotoBottom()

		return m, waitForDebuggerOutput(m.debugger.Output)

	case messages.DebuggerCommandReceived:
		label := commandLabelStyle.Render("[command] ")
		m.content += "\n" + label + string(msg)
		m.viewport.SetContent(m.content)
		m.viewport.GotoBottom()

		return m, waitForDebuggerOutput(m.debugger.Output)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.viewport.Width = m.width
		m.viewport.Height = m.height

		return m, nil

	case tea.KeyMsg:
		var cmd tea.Cmd

		if !m.IsFocused {
			return m, nil
		}

		if !m.commandInput.IsFocused && msg.String() == "i" {
			m.commandInput, cmd = m.commandInput.Update(messages.WindowFocused(m.ID))
			return m, cmd
		}

		if m.commandInput.IsFocused {
			m.commandInput, cmd = m.commandInput.Update(msg)
			return m, cmd
		}

		m.viewport, cmd = m.viewport.Update(msg)

		return m, cmd
	}
	return m, nil
}

func (m Model) View() string {
	return lipgloss.JoinVertical(lipgloss.Top,
		outputContentStyle.Render(m.viewport.View()),
		m.commandInput.View(),
	)
}
