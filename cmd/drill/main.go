package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/andersonjoseph/drill/internal/components/breakpoints"
	"github.com/andersonjoseph/drill/internal/components/localvariables"
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
	code         sourcecode.Model
	currentIndex int
	debugger     *debugger.Debugger
	logs         []string
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.handleResize(msg)
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		if msg.String() == "n" {
			_, err := m.debugger.Client.Next()
			if err != nil {
				fmt.Println("Error getting debugger state:", err)
				os.Exit(1)
			}
			m.updateContent()
			return m, nil
		}

		if msg.String() == "c" {
			<-m.debugger.Client.Continue()
			m.updateContent()
			return m, nil
		}

		if msg.String() == "r" {
			m.debugger.Client.Restart(false)
			m.updateContent()
			return m, nil
		}

		if msg.String() == "t" {
			m.sidebar.breakpoints, _ = m.sidebar.breakpoints.Update(messages.ToggleBreakpoint{})
			return m, nil
		}

		if msg.String() == "d" {
			m.sidebar.breakpoints, _ = m.sidebar.breakpoints.Update(messages.ClearBreakpoint{})
			return m, nil
		}

		if msg.String() == "a" {
			m.sidebar.breakpoints, _ = m.sidebar.breakpoints.Update(messages.CreateBreakpointNow{})
			return m, nil
		}

		m.sidebar.localVariables, cmd = m.sidebar.localVariables.Update(msg)
		cmds = append(cmds, cmd)

		m.sidebar.breakpoints, cmd = m.sidebar.breakpoints.Update(msg)
		cmds = append(cmds, cmd)

		return m, tea.Batch(cmds...)
	}

	return m, nil
}

func (m model) View() string {
	if m.sidebar.localVariables.Error != nil {
		return m.sidebar.localVariables.Error.Error()
	}
	if m.sidebar.localVariables.Error != nil {
		return m.code.Error.Error()
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
			m.code.View(),
		),
	)
}

func (m *model) updateContent() {
	m.sidebar.localVariables, _ = m.sidebar.localVariables.Update(messages.UpdateContent{})
	m.sidebar.breakpoints, _ = m.sidebar.breakpoints.Update(messages.UpdateContent{})

	m.code, _ = m.code.Update(messages.UpdateContent{})
}

func (m *model) handleResize(msg tea.WindowSizeMsg) {
	sidebarWidth, sidebarHeight := getSideBarSize(msg.Width, msg.Height)

	m.sidebar.localVariables.Width = sidebarWidth
	m.sidebar.localVariables.Height = sidebarHeight
	m.sidebar.localVariables, _ = m.sidebar.localVariables.Update(msg)

	m.sidebar.breakpoints.Width = sidebarWidth
	m.sidebar.breakpoints.Height = sidebarHeight
	m.sidebar.breakpoints, _ = m.sidebar.breakpoints.Update(msg)

	m.code.Height = max(msg.Height-2, 5)
	m.code.Width = (msg.Width - sidebarWidth) - 4

	m.code, _ = m.code.Update(msg)
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
		code: sourcecode.New("Source Code", debugger),
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
