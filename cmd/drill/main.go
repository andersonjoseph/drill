package main

import (
	"fmt"
	"os"

	"github.com/andersonjoseph/drill/internal/components/localvariables"
	"github.com/andersonjoseph/drill/internal/components/sourcecode"
	"github.com/andersonjoseph/drill/internal/debugger"
	"github.com/andersonjoseph/drill/internal/messages"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type sidebar struct {
	localVariables localvariables.Model
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

		m.sidebar.localVariables, cmd = m.sidebar.localVariables.Update(msg)
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

	sidebar := m.sidebar.localVariables.View()
	mainContent := m.code.View()

	return lipgloss.JoinVertical(
		lipgloss.Top,
		lipgloss.JoinHorizontal(lipgloss.Top, sidebar, mainContent),
	)
}

func (m *model) updateContent() {
	m.sidebar.localVariables, _ = m.sidebar.localVariables.Update(messages.UpdateContent{})
	m.code, _ = m.code.Update(messages.UpdateContent{})
}

func (m *model) handleResize(msg tea.WindowSizeMsg) {
	m.sidebar.localVariables, _ = m.sidebar.localVariables.Update(msg)
	m.code, _ = m.code.Update(msg)
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
			localVariables: localvariables.New(1, "Local Variables", debugger),
		},
		code: sourcecode.New(debugger),
	}

	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
