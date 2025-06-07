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
)

type Model struct {
	ID        int
	IsFocused bool
	title     string
	content   string
	Width     int
	Height    int
	Error     error
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

func waitForStdout(c chan string) tea.Cmd {
	return func() tea.Msg {
		return messages.DebuggerStdout(<-c)
	}
}

func waitForStderr(c chan string) tea.Cmd {
	return func() tea.Msg {
		return messages.DebuggerStderr(<-c)
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		waitForStdout(m.debugger.Stdout),
		waitForStderr(m.debugger.Stderr),
	)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.IsFocused:
		m.IsFocused = bool(msg)
		return m, nil

	case messages.Restart:
		m.content = ""
		m.viewport.SetContent(m.content)
		return m, nil

	case messages.DebuggerStdout:
		m.content += "\n" + string(msg)
		m.viewport.SetContent(m.content)
		m.viewport.ScrollDown(1)

		return m, waitForStdout(m.debugger.Stdout)

	case messages.DebuggerStderr:
		m.content += "\n" + string(msg)
		m.viewport.SetContent(m.content)
		m.viewport.ScrollDown(1)

		return m, waitForStderr(m.debugger.Stderr)

	case tea.WindowSizeMsg:
		m.viewport.Width = m.Width
		m.viewport.Height = m.Height

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
	topBorder := "┌" + title + strings.Repeat("─", max(m.Width-len(title), 1)) + "┐"

	scrollPercent := fmt.Sprintf("%d%%", int(m.viewport.ScrollPercent()*100))
	bottomBorder := "└" + strings.Repeat("─", max(m.Width-len(scrollPercent), 1)) + scrollPercent + "┘"

	return lipgloss.JoinVertical(
		lipgloss.Top,
		style.Render(topBorder),
		style.
			Border(lipgloss.NormalBorder()).
			BorderTop(false).
			BorderBottom(false).
			BorderForeground(style.GetForeground()).
			Height(m.Height).
			Width(m.Width).
			Render(listDefaultStyle.Render(m.viewport.View())),
		style.Render(bottomBorder),
	)
}
