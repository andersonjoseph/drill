package output

import (
	"fmt"
	"strings"

	"github.com/andersonjoseph/drill/internal/components"
	"github.com/andersonjoseph/drill/internal/debugger"
	"github.com/andersonjoseph/drill/internal/messages"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	listFocusedStyle lipgloss.Style = lipgloss.NewStyle().Foreground(components.ColorGreen)
	listDefaultStyle lipgloss.Style = lipgloss.NewStyle().Foreground(components.ColorWhite)

	stdoutLabelStyle lipgloss.Style = lipgloss.NewStyle().Foreground(components.ColorGrey)
	stderrLabelStyle lipgloss.Style = lipgloss.NewStyle().Foreground(components.ColorOrange)
)

type Model struct {
	ID        int
	IsFocused bool
	title     string
	content   string
	width     int
	height    int
	viewport  viewport.Model
	debugger  *debugger.Debugger
}

func New(id int, title string, d *debugger.Debugger) Model {
	m := Model{
		ID:       id,
		title:    title,
		debugger: d,
		content:  "",
		viewport: viewport.New(30, 5),
	}

	return m
}

func waitForDebuggerOutput(c chan debugger.Output) tea.Cmd {
	return func() tea.Msg {
		o := <-c
		if o.Source == debugger.SourceStderr {
			return messages.DebuggerStderrReceived(o.Content)
		}
		return messages.DebuggerStdoutReceived(o.Content)
	}
}

func (m Model) Init() tea.Cmd {
	return waitForDebuggerOutput(m.debugger.Output)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.WindowFocused:
		m.IsFocused = int(msg) == m.ID
		return m, nil

	case messages.DebuggerRestarted:
		m.content = ""
		m.viewport.SetContent(m.content)
		return m, nil

	case messages.DebuggerStdoutReceived:
		label := stdoutLabelStyle.Render("[stdout] ")
		m.content += "\n" + label + string(msg)
		m.viewport.SetContent(m.content)
		m.viewport.ScrollDown(1)

		return m, waitForDebuggerOutput(m.debugger.Output)

	case messages.DebuggerStderrReceived:
		label := stderrLabelStyle.Render("[stderr] ")
		m.content += "\n" + label + string(msg)
		m.viewport.SetContent(m.content)
		m.viewport.ScrollDown(1)

		return m, waitForDebuggerOutput(m.debugger.Output)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.viewport.Width = m.width
		m.viewport.Height = m.height

		return m, nil

	case tea.KeyMsg:
		if !m.IsFocused {
			return m, nil
		}

		if msg.String() == "c" {
			m.content = ""
			m.viewport.SetContent(m.content)
			return m, nil
		}

		var cmd tea.Cmd
		var cmds []tea.Cmd

		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)

		return m, tea.Batch(cmds...)
	}
	return m, nil
}

func (m Model) View() string {
	var style lipgloss.Style
	if m.IsFocused {
		style = listFocusedStyle
	} else {
		style = listDefaultStyle
	}

	title := fmt.Sprintf("[%d] %s", m.ID, m.title)
	topBorder := "┌" + title + strings.Repeat("─", max(m.width-len(title), 1)) + "┐"

	scrollPercent := fmt.Sprintf("%d%%", int(m.viewport.ScrollPercent()*100))
	bottomBorder := "└" + strings.Repeat("─", max(m.width-len(scrollPercent), 1)) + scrollPercent + "┘"

	return lipgloss.JoinVertical(
		lipgloss.Top,
		style.Render(topBorder),
		style.
			Border(lipgloss.NormalBorder()).
			BorderTop(false).
			BorderBottom(false).
			BorderForeground(style.GetForeground()).
			Height(m.height).
			Width(m.width).
			Render(listDefaultStyle.Render(m.viewport.View())),
		style.Render(bottomBorder),
	)
}
