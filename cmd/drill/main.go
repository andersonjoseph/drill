package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/andersonjoseph/drill/internal/components/breakpoints"
	"github.com/andersonjoseph/drill/internal/components/localvariables"
	"github.com/andersonjoseph/drill/internal/components/output"
	"github.com/andersonjoseph/drill/internal/components/sourcecode"
	"github.com/andersonjoseph/drill/internal/debugger"
	"github.com/andersonjoseph/drill/internal/messages"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type sidebar struct {
	localVariables localvariables.Model
	breakpoints    breakpoints.Model
}

type model struct {
	sidebar      sidebar
	sourceCode   sourcecode.Model
	output       output.Model
	currentIndex int
	debugger     *debugger.Debugger
	logs         []string
	error        error
}

func (m model) Init() tea.Cmd {
	return m.output.Init()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case messages.UpdateContent:
		m.updateContent()

	case tea.WindowSizeMsg:
		m.handleResize(msg)
		return m, nil

	case messages.DebuggerOutput:
		m.output, cmd = m.output.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		if msg.String() == "r" {
			m.error = nil
			_, err := m.debugger.Client.Restart(false)
			if err != nil {
				m.error = fmt.Errorf("error restarting debugger: %w", err)
				return m, nil
			}
			m.updateContent()
			m.output, cmd = m.output.Update(messages.Restart{})

			return m, cmd
		}

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

	return m, nil
}

func buildErrMessage(err error) string {
	if strings.Contains(err.Error(), "has exited with status 0") {
		return "debug session ended press r to reset or q to quit"
	}

	return err.Error()
}

func (m model) View() string {
	if m.error != nil {
		return buildErrMessage(m.error)
	}

	return lipgloss.JoinVertical(
		lipgloss.Top,
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			lipgloss.JoinVertical(
				lipgloss.Top,
				m.sidebar.localVariables.View(),
				m.sidebar.breakpoints.View(),
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

	if err := m.sidebar.localVariables.Error; err != nil {
		m.error = err
		m.sidebar.localVariables.Error = nil
	}
	if err := m.sidebar.breakpoints.Error; err != nil {
		m.error = err
		m.sidebar.breakpoints.Error = nil
	}

	m.sourceCode, _ = m.sourceCode.Update(messages.UpdateContent{})
	if err := m.sourceCode.Error; err != nil {
		m.error = err
		m.sourceCode.Error = nil
	}
}

func (m *model) handleResize(msg tea.WindowSizeMsg) {
	sidebarWidth, sidebarHeight := getSideBarSize(msg.Width, msg.Height)

	m.sidebar.localVariables.Width = sidebarWidth
	m.sidebar.localVariables.Height = sidebarHeight
	m.sidebar.localVariables, _ = m.sidebar.localVariables.Update(msg)

	m.sidebar.breakpoints.Width = sidebarWidth
	m.sidebar.breakpoints.Height = sidebarHeight
	m.sidebar.breakpoints, _ = m.sidebar.breakpoints.Update(msg)

	m.sourceCode.Height = max((msg.Height)-10, 5)
	m.sourceCode.Width = (msg.Width - sidebarWidth) - 4
	m.sourceCode, _ = m.sourceCode.Update(msg)

	m.output.Height = max((msg.Height-m.sourceCode.Height)-5, 2)
	m.output.Width = (msg.Width - sidebarWidth) - 4
	m.output, _ = m.output.Update(msg)
}

func getSideBarSize(w, h int) (int, int) {
	w = w / 2
	if w >= 50 {
		w = 50
	} else if w <= 20 {
		w = 20
	}

	h = h / 3
	if h >= 15 {
		h = 15
	} else if h <= 3 {
		h = 3
	}

	return w, h
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
		debugger: debugger,
		sidebar: sidebar{
			localVariables: localvariables.New(1, debugger),
			breakpoints:    breakpoints.New(2, debugger),
		},
		sourceCode: sourcecode.New(3, "Source Code", debugger),
		output:     output.New(4, "Output", debugger),
	}

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
