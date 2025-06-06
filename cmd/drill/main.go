package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/andersonjoseph/drill/internal/components"
	"github.com/andersonjoseph/drill/internal/components/breakpoints"
	"github.com/andersonjoseph/drill/internal/components/localvariables"
	"github.com/andersonjoseph/drill/internal/components/output"
	"github.com/andersonjoseph/drill/internal/components/sourcecode"
	"github.com/andersonjoseph/drill/internal/debugger"
	"github.com/andersonjoseph/drill/internal/messages"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var warningStyle lipgloss.Style = lipgloss.NewStyle().Foreground(components.ColorOrange).BorderForeground(components.ColorOrange)

type sidebar struct {
	localVariables localvariables.Model
	breakpoints    breakpoints.Model
	width          int
	height         int
}

func (s *sidebar) calcSize(w, h int) {
	w = w / 2
	if w >= 50 {
		w = 50
	} else if w <= 20 {
		w = 20
	}
	s.width = w

	h = h / 4
	if h >= 15 {
		h = 15
	} else if h <= 3 {
		h = 3
	}
	s.height = h
}

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
	return m.output.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case messages.Restart:
		m.updateContent()
		return m, nil

	case messages.FocusedWindow:
		m.focusedWindow = int(msg)
		return m, nil

	case messages.UpdateContent:
		m.updateContent()
		return m, nil

	case tea.WindowSizeMsg:
		m.handleResize(msg)
		return m, nil

	case messages.DebuggerOutput:
		m.output, cmd = m.output.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		if m.focusedWindow != 0 && (msg.String() == "q" || msg.String() == "ctrl+c") {
			return m, tea.Quit
		}

		if m.focusedWindow != 0 {
			if id, err := strconv.Atoi(msg.String()); err == nil {
				m.focusedWindow = id

				m.sidebar.localVariables, cmd = m.sidebar.localVariables.Update(messages.IsFocused(m.focusedWindow == m.sidebar.localVariables.ID))
				cmds = append(cmds, cmd)

				m.sidebar.breakpoints, cmd = m.sidebar.breakpoints.Update(messages.IsFocused(m.focusedWindow == m.sidebar.breakpoints.ID))
				cmds = append(cmds, cmd)

				m.sourceCode, cmd = m.sourceCode.Update(messages.IsFocused(m.focusedWindow == m.sourceCode.ID))
				cmds = append(cmds, cmd)

				m.output, cmd = m.output.Update(messages.IsFocused(m.focusedWindow == m.output.ID))
				cmds = append(cmds, cmd)

				m.bubbleUpComponentErrors()
				return m, tea.Batch(cmds...)
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
	}

	return m, nil
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

func parseEntryBreakpoint(bp string) (string, int, error) {
	breakpointAttrs := strings.Split(bp, ":")

	filename := breakpointAttrs[0]
	line, err := strconv.Atoi(breakpointAttrs[1])

	return filename, line, err
}

func main() {
	debugger, err := debugger.New()
	if err != nil {
		fmt.Println("Error creating debugger", err)
		os.Exit(1)
	}
	defer debugger.Client.Disconnect(false)
	m := model{
		focusedWindow: 1,
		debugger:      debugger,
		sidebar: sidebar{
			localVariables: localvariables.New(1, debugger),
			breakpoints:    breakpoints.New(2, debugger),
		},
		sourceCode: sourcecode.New(3, "Source Code", debugger),
		output:     output.New(4, "Output", debugger),
	}
	m.sidebar.localVariables.IsFocused = true

	var bp string
	var autoContinue bool
	flag.StringVar(&bp, "bp", "", "create a breakpoint")
	flag.BoolVar(&autoContinue, "c", false, "create a breakpoint")

	flag.Parse()

	if bp != "" {
		filename, line, err := parseEntryBreakpoint(bp)
		if err != nil {
			fmt.Println("Error parsing breakpoint:", err)
			os.Exit(1)
		}

		_, err = debugger.CreateBreakpoint(filename, line)
		if err != nil {
			fmt.Println("Error parsing breakpoint:", err)
			os.Exit(1)
		}
		if autoContinue {
			<-debugger.Client.Continue()
			m.updateContent()
		}
	}

	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
