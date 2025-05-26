package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/andersonjoseph/drill/internal/components/sourcecode"
	"github.com/andersonjoseph/drill/internal/components/variablelist"
	"github.com/andersonjoseph/drill/internal/messages"
	"github.com/andersonjoseph/drill/internal/types"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
)

type sidebar struct {
	localVariables tea.Model
}

type model struct {
	sidebar     sidebar
	code tea.Model
	currentIndex int
	debugger debugger
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

		if msg.String() == "n" {
			_, err := m.debugger.client.Next()
			if err != nil {
				fmt.Println("Error getting debugger state:", err)
				os.Exit(1)
			}

			fileContent := m.debugger.getCurrentFileContent()
			m.code, cmd = m.code.Update(messages.NewCodeContent(fileContent))

			localVariables := m.debugger.getLocalVariables()

			m.sidebar.localVariables, cmd =  m.sidebar.localVariables.Update(messages.NewVariables(localVariables))
			return m, cmd
		}

		if msg.String() == "c" {
			<-m.debugger.client.Continue()

			fileContent := m.debugger.getCurrentFileContent()
			m.code, cmd = m.code.Update(messages.NewCodeContent(fileContent))

			localVariables := m.debugger.getLocalVariables()

			m.sidebar.localVariables, cmd =  m.sidebar.localVariables.Update(messages.NewVariables(localVariables))
			return m, cmd
		}
	}

	m.sidebar.localVariables, cmd = m.sidebar.localVariables.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	sidebar := m.sidebar.localVariables.View()

	mainContent := m.code.View()
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, mainContent)
}

type debugger struct {
	client *rpc2.RPCClient
	ready chan string
}

func New() (debugger, error) {
	d := debugger{
		ready: make(chan string),
	}
	d.startProcess()

	select {
	case addr := <- d.ready:
		fmt.Printf("addr: %v\n", addr)
		d.client = rpc2.NewClient(addr)
	case <-time.After(time.Second * 10):
		return debugger{}, errors.New("timeout")
	}

	return d, nil
}

func (d debugger) startProcess() {
    cmd := exec.Command("dlv", "debug", "--headless")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Println("Error getting local variables:", err)
		os.Exit(1)
	}

	cmd.Start()
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), "listening") {
				println("debugger ready")
				d.ready <- regexp.MustCompile(`\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?):\d{1,5}\b`).FindString(scanner.Text())
			}
		}
	}()
}

func (d debugger) getCurrentFileContent() string {
	state, err := d.client.GetState()
	if err != nil {
		fmt.Println("Error getting debugger state:", err)
		os.Exit(1)
	}

	filename := state.CurrentThread.File
	breakpointLine := state.CurrentThread.Line

	f, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Error opening file: %s: %v", filename, err)
		os.Exit(1)
	}

	offset := 10

	scanner := bufio.NewScanner(f)
	currentLine := 0
	startLine := max(0, breakpointLine-offset)
	endLine := breakpointLine+offset

	lines := strings.Builder{}

	for scanner.Scan() && currentLine < endLine {
		currentLine++
		if currentLine < startLine {
			continue
		}
		if currentLine == breakpointLine {
			lines.WriteString("=>")
		}
		lines.WriteString(scanner.Text() + "\n")
	}

	return lines.String()
}

func (d debugger) getLocalVariables() []types.Variable {
	state, err := d.client.GetState()

	if err != nil {
		fmt.Println("Error getting state:", err)
		os.Exit(1)
	}

	vars, err := d.client.ListLocalVariables(
		api.EvalScope{
		GoroutineID: state.CurrentThread.GoroutineID,
	}, api.LoadConfig{})

	if err != nil {
		fmt.Println("Error getting local variables:", err)
		os.Exit(1)
	}

	res := make([]types.Variable, len(vars))

	for i := range vars {
		res[i] = types.Variable{
			Name: vars[i].Name,
			Value: vars[i].Value,
		}
	}

	return res
}

func main() {
	debugger, err := New()
	if err != nil {
		fmt.Println("Error getting local variables:", err)
		os.Exit(1)
	}
	defer debugger.client.Disconnect(false)

	m := model {
		debugger: debugger,
		sidebar: sidebar{
			localVariables: variablelist.New("Local Variables", 1),
		},
		code: sourcecode.New(debugger.getCurrentFileContent()),
	}

	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
