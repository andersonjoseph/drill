package output

import (
	"fmt"
	"strconv"
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
	id        int
	isFocused bool
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
		id:       id,
		title:    title,
		debugger: d,
		content:  "",
		viewport: viewport.New(30, 5),
	}

	return m
}

func waitForOutput(c chan string) tea.Cmd {
	return func() tea.Msg {
		return messages.DebuggerOutput(<-c)
	}
}

func (m Model) Init() tea.Cmd {
	return waitForOutput(m.debugger.Output)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.Restart:
		m.content = ""
		m.viewport.SetContent(m.content)
		return m, nil

	case messages.DebuggerOutput:
		m.content += "\n" + string(msg)
		m.viewport.SetContent(m.content)
		m.viewport.ScrollDown(1)

		return m, waitForOutput(m.debugger.Output)

	case tea.WindowSizeMsg:
		m.viewport.Width = m.Width
		m.viewport.Height = m.Height

		return m, nil

	case tea.KeyMsg:
		if id, err := strconv.Atoi(msg.String()); err == nil {
			m.isFocused = id == m.id
			return m, nil
		}
		if !m.isFocused {
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
	if m.isFocused {
		style = listFocusedStyle
	} else {
		style = listDefaultStyle
	}

	title := fmt.Sprintf("%s [%d] ", m.title, m.id)
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
