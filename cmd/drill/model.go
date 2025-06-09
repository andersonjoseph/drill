package main

import (
	"strconv"
	"strings"

	"github.com/andersonjoseph/drill/internal/components"
	"github.com/andersonjoseph/drill/internal/components/output"
	"github.com/andersonjoseph/drill/internal/components/sourcecode"
	"github.com/andersonjoseph/drill/internal/debugger"
	"github.com/andersonjoseph/drill/internal/messages"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var warningStyle lipgloss.Style = lipgloss.NewStyle().Foreground(components.ColorOrange).BorderForeground(components.ColorOrange)

type model struct {
	sidebar       sidebar
	sourceCode    sourcecode.Model
	output        output.Model
	debugger      *debugger.Debugger
	logs          []string
	error         error
	focusedWindow int
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			return messages.UpdateContent{}

		},
		func() tea.Msg {
			return messages.FocusedWindow(3)

		},
		m.output.Init(),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case messages.Error:
		m.error = msg
		return m, nil

	case messages.FocusedWindow:
		return m, m.updateFocus(int(msg))

	case messages.UpdateContent:
		m.sidebar.localVariables, cmd = m.sidebar.localVariables.Update(messages.UpdateContent{})
		cmds = append(cmds, cmd)

		m.sidebar.breakpoints, cmd = m.sidebar.breakpoints.Update(messages.UpdateContent{})
		cmds = append(cmds, cmd)

		m.sourceCode, cmd = m.sourceCode.Update(messages.UpdateContent{})
		cmds = append(cmds, cmd)

		return m, tea.Batch(cmds...)

	case tea.WindowSizeMsg:
		return m, m.handleResize(msg)

	case tea.KeyMsg:
		if m.focusedWindow != 0 && (msg.String() == "q" || msg.String() == "ctrl+c") {
			return m, tea.Quit
		}

		if m.focusedWindow != 0 {
			if focusedWindow, err := strconv.Atoi(msg.String()); err == nil {
				m.updateFocus(focusedWindow)
			}
		}

		m.sidebar.localVariables, cmd = m.sidebar.localVariables.Update(msg)
		cmds = append(cmds, cmd)

		m.sidebar.breakpoints, cmd = m.sidebar.breakpoints.Update(msg)
		cmds = append(cmds, cmd)

		m.sourceCode, cmd = m.sourceCode.Update(msg)
		cmds = append(cmds, cmd)

		m.output, cmd = m.output.Update(msg)
		cmds = append(cmds, cmd)

		m.error = nil
		return m, tea.Batch(cmds...)

	default:
		m.sidebar.localVariables, cmd = m.sidebar.localVariables.Update(msg)
		cmds = append(cmds, cmd)

		m.sidebar.breakpoints, cmd = m.sidebar.breakpoints.Update(msg)
		cmds = append(cmds, cmd)

		m.sourceCode, cmd = m.sourceCode.Update(msg)
		cmds = append(cmds, cmd)

		m.output, cmd = m.output.Update(msg)
		cmds = append(cmds, cmd)

		return m, tea.Batch(cmds...)
	}
}

func (m *model) updateFocus(focusedWindow int) tea.Cmd {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	m.focusedWindow = focusedWindow

	m.sidebar.localVariables, cmd = m.sidebar.localVariables.Update(messages.IsFocused(m.focusedWindow == m.sidebar.localVariables.ID))
	cmds = append(cmds, cmd)

	m.sidebar.breakpoints, cmd = m.sidebar.breakpoints.Update(messages.IsFocused(m.focusedWindow == m.sidebar.breakpoints.ID))
	cmds = append(cmds, cmd)

	m.sourceCode, cmd = m.sourceCode.Update(messages.IsFocused(m.focusedWindow == m.sourceCode.ID))
	cmds = append(cmds, cmd)

	m.output, cmd = m.output.Update(messages.IsFocused(m.focusedWindow == m.output.ID))
	cmds = append(cmds, cmd)

	return tea.Batch(cmds...)
}

func (m model) viewErrMessage() string {
	//TODO: move this to its own component
	if m.error == nil {
		return ""
	}

	msg := m.error.Error()
	style := warningStyle

	if strings.Contains(msg, "has exited with status 0") {
		msg = "debug session ended press r to reset or q to quit"
	}

	if strings.Contains(msg, "error evaluating expression:") {
		msg = "breakpoint condition failed:" + strings.Split(msg, "error evaluating expression:")[1]
	}

	title := "Attention"
	topBorder := "┌" + title + strings.Repeat("─", max(30-len(title), 1)) + "┐"
	return lipgloss.JoinVertical(
		lipgloss.Top,
		style.Render(topBorder),
		style.
			Border(lipgloss.NormalBorder()).
			Width(30).
			BorderTop(false).
			BorderForeground().
			Render(msg),
	)
}

func (m model) View() string {
	return lipgloss.JoinVertical(
		lipgloss.Top,
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			lipgloss.JoinVertical(
				lipgloss.Top,
				m.sidebar.localVariables.View(),
				m.sidebar.breakpoints.View(),
				m.viewErrMessage(),
			),
			lipgloss.JoinVertical(
				lipgloss.Top,
				m.sourceCode.View(),
				m.output.View(),
			),
		),
	)
}

func (m *model) handleResize(msg tea.WindowSizeMsg) tea.Cmd {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	sidebarWidth, sidebarHeight := m.sidebar.calcSize(msg.Width, msg.Height)

	m.sidebar.localVariables, cmd = m.sidebar.localVariables.Update(tea.WindowSizeMsg{Width: sidebarWidth, Height: sidebarHeight})
	cmds = append(cmds, cmd)

	m.sidebar.breakpoints, cmd = m.sidebar.breakpoints.Update(tea.WindowSizeMsg{Width: sidebarWidth, Height: sidebarHeight})
	cmds = append(cmds, cmd)

	sourceCodeHeight := max((msg.Height)-10, 5)
	sourceCodeWidth := (msg.Width - sidebarWidth) - 4

	m.sourceCode, cmd = m.sourceCode.Update(tea.WindowSizeMsg{Width: sourceCodeWidth, Height: sourceCodeHeight})
	cmds = append(cmds, cmd)

	outputHeight := max((msg.Height-sourceCodeHeight)-5, 2)
	outputWidth := (msg.Width - sidebarWidth) - 4
	m.output, cmd = m.output.Update(tea.WindowSizeMsg{Width: outputWidth, Height: outputHeight})
	cmds = append(cmds, cmd)

	return tea.Batch(cmds...)
}
