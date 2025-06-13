package window

import (
	"fmt"
	"strings"

	"github.com/andersonjoseph/drill/internal/components"
	"github.com/andersonjoseph/drill/internal/messages"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	windowFocusedStyle = lipgloss.NewStyle().Foreground(components.ColorGreen)
	windowDefaultStyle = lipgloss.NewStyle()
)

type Model struct {
	ID        int
	Title     string
	IsFocused bool
	Width     int
	Height    int

	child tea.Model
}

func New(id int, title string, child tea.Model) Model {
	return Model{
		ID:    id,
		Title: title,
		child: child,
	}
}

func (m Model) Init() tea.Cmd {
	return m.child.Init()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case messages.WindowFocused:
		m.IsFocused = (int(msg) == m.ID)

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height

		childWidth := m.Width - 2
		innerHeight := m.Height - 2

		m.child, cmd = m.child.Update(tea.WindowSizeMsg{
			Width:  childWidth,
			Height: innerHeight,
		})
		return m, cmd
	}

	m.child, cmd = m.child.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	var style lipgloss.Style
	if m.IsFocused {
		style = windowFocusedStyle
	} else {
		style = windowDefaultStyle
	}

	title := style.Render(fmt.Sprintf("[%d] %s", m.ID, m.Title))
	titleWidth := lipgloss.Width(title)
	topBorder := style.Render("┌") + title + style.Render(strings.Repeat("─", max(m.Width-titleWidth, 0))) + style.Render("┐")

	return lipgloss.JoinVertical(lipgloss.Top,
		topBorder,
		style.
			Width(m.Width).
			Border(lipgloss.NormalBorder()).
			BorderForeground(style.GetForeground()).
			BorderTop(false).
			Render(m.child.View()),
	)
}
