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
			return messages.FocusedWindow(3)

		},
		m.output.Init(),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case messages.FocusedWindow:
		cmd = m.updateFocus(int(msg))
		return m, cmd

	case messages.UpdateContent:
		m.updateContent()
		return m, nil

	case tea.WindowSizeMsg:
		m.handleResize(msg)
		return m, nil

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
		m.bubbleUpComponentErrors()
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

		m.bubbleUpComponentErrors()
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

	m.bubbleUpComponentErrors()
	return tea.Batch(cmds...)
}

func (m *model) bubbleUpComponentErrors() {
	if err := m.sidebar.localVariables.Error; err != nil {
		m.error = err
		m.sidebar.localVariables.Error = nil
		return
	}
	if err := m.sidebar.breakpoints.Error; err != nil {
		m.error = err
		m.sidebar.breakpoints.Error = nil
		return
	}
	if err := m.sourceCode.Error; err != nil {
		m.error = err
		m.sourceCode.Error = nil
		return
	}
}

func (m model) viewErrMessage() string {
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
	topBorder := "┌" + title + strings.Repeat("─", max(m.sidebar.width-len(title), 1)) + "┐"

	return lipgloss.JoinVertical(
		lipgloss.Top,
		style.Render(topBorder),
		style.
			Border(lipgloss.NormalBorder()).
			Width(m.sidebar.width).
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

func (m *model) updateContent() {
	m.sidebar.localVariables, _ = m.sidebar.localVariables.Update(messages.UpdateContent{})
	m.sidebar.breakpoints, _ = m.sidebar.breakpoints.Update(messages.UpdateContent{})
	m.sourceCode, _ = m.sourceCode.Update(messages.UpdateContent{})

	m.bubbleUpComponentErrors()
}

func (m *model) handleResize(msg tea.WindowSizeMsg) {
	m.sidebar.calcSize(msg.Width, msg.Height)

	m.sidebar.localVariables.Width = m.sidebar.width
	m.sidebar.localVariables.Height = m.sidebar.height
	m.sidebar.localVariables, _ = m.sidebar.localVariables.Update(msg)

	m.sidebar.breakpoints.Width = m.sidebar.width
	m.sidebar.breakpoints.Height = m.sidebar.height
	m.sidebar.breakpoints, _ = m.sidebar.breakpoints.Update(msg)

	m.sourceCode.Height = max((msg.Height)-10, 5)
	m.sourceCode.Width = (msg.Width - m.sidebar.width) - 4
	m.sourceCode, _ = m.sourceCode.Update(msg)

	m.output.Height = max((msg.Height-m.sourceCode.Height)-5, 2)
	m.output.Width = (msg.Width - m.sidebar.width) - 4
	m.output, _ = m.output.Update(msg)
}
