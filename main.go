package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/andersonjoseph/drill/internal/components/variablelist"
	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	sections     []tea.Model
	currentIndex int
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	for i, sect := range m.sections {
		m.sections[i], cmd = sect.Update(msg)
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	views := make([]string, len(m.sections))

	for i, sect := range m.sections {
		views[i] = sect.View()
	}

	return strings.Join(views, "\n")
}

func main() {
	localVariables := []variablelist.Variable{
		{
			Name:  "age",
			Value: "2112312",
		},
	}

	globalVariables := []variablelist.Variable{
		{
			Name:  "user",
			Value: "andersonjoseph",
		},
		{
			Name:  "ip",
			Value: "666.666.666",
		},
	}

	watchVariables := []variablelist.Variable{}

	m := model{
		sections: []tea.Model{
			variablelist.New("Local Variables", 1, localVariables),
			variablelist.New("Global Variables", 2, globalVariables),
			variablelist.New("Watch", 3, watchVariables),
		},
	}

	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
