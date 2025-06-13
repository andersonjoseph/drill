package main

import (
	"strconv"

	"github.com/andersonjoseph/drill/internal/components/window"
	"github.com/andersonjoseph/drill/internal/debugger"
	"github.com/andersonjoseph/drill/internal/messages"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	sidebar          []window.Model
	sourceCode       window.Model
	output           window.Model
	debugger         *debugger.Debugger
	logs             []string
	textInputFocused bool
	focusedWindow    int
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			return messages.RefreshContent{}

		},
		func() tea.Msg {
			return messages.WindowFocused(4)
		},
		m.output.Init(),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case messages.Error:
		m.sidebar[len(m.sidebar)-1], cmd = m.sidebar[len(m.sidebar)-1].Update(msg)
		return m, cmd

	case messages.WindowFocused:
		m.focusedWindow = int(msg)

	case messages.TextInputFocused:
		m.textInputFocused = bool(msg)
		return m, nil

	case tea.WindowSizeMsg:
		return m, m.handleResize(msg)

	case tea.KeyMsg:
		if !m.textInputFocused && (msg.String() == "q" || msg.String() == "ctrl+c") {
			return m, tea.Quit
		}

		if !m.textInputFocused && msg.String() != "0" {
			if focusedWindow, err := strconv.Atoi(msg.String()); err == nil {
				return m, func() tea.Msg {
					return messages.WindowFocused(focusedWindow)
				}
			}
		}

		m.sidebar[len(m.sidebar)-1].Update(messages.Error(nil))
	}

	for i := range m.sidebar {
		m.sidebar[i], cmd = m.sidebar[i].Update(msg)
		cmds = append(cmds, cmd)
	}

	m.sourceCode, cmd = m.sourceCode.Update(msg)
	cmds = append(cmds, cmd)

	m.output, cmd = m.output.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	return lipgloss.JoinVertical(
		lipgloss.Top,
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			lipgloss.JoinVertical(
				lipgloss.Top,
				m.sidebar[0].View(),
				m.sidebar[1].View(),
				m.sidebar[2].View(),
				//m.sidebar[3].View(),
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
	const (
		sidebarRatio      = 0.3
		minSidebarWidth   = 20
		maxSidebarWidth   = 50
		mainPanelVPadding = 4
		mainPanelHPadding = 4
	)

	// --- Sidebar Calculations ---
	sidebarWidth := int(float64(msg.Width) * sidebarRatio)
	if sidebarWidth < minSidebarWidth {
		sidebarWidth = minSidebarWidth
	}
	if sidebarWidth > maxSidebarWidth {
		sidebarWidth = maxSidebarWidth
	}

	sidebarAvailableHeight := msg.Height - 4 // e.g., 1px border per component
	sidebarComponentHeight := sidebarAvailableHeight / 4

	// --- Main Panel Calculations (Source Code + Output) ---
	mainPanelWidth := msg.Width - sidebarWidth - mainPanelHPadding
	mainPanelAvailableHeight := msg.Height - mainPanelVPadding

	// Let's give the source code 70% of the available vertical space
	// and the output the remaining 30%.
	sourceCodeHeight := int(float64(mainPanelAvailableHeight) * 0.7)
	outputHeight := mainPanelAvailableHeight - sourceCodeHeight

	// --- Update all components with their new sizes ---
	var cmds []tea.Cmd
	var cmd tea.Cmd

	for i := range m.sidebar {
		m.sidebar[i], cmd = m.sidebar[i].Update(tea.WindowSizeMsg{Width: sidebarWidth, Height: sidebarComponentHeight})
		cmds = append(cmds, cmd)
	}

	m.sourceCode, cmd = m.sourceCode.Update(tea.WindowSizeMsg{Width: mainPanelWidth, Height: sourceCodeHeight})
	cmds = append(cmds, cmd)

	m.output, cmd = m.output.Update(tea.WindowSizeMsg{Width: mainPanelWidth, Height: outputHeight})
	cmds = append(cmds, cmd)

	return tea.Batch(cmds...)
}
