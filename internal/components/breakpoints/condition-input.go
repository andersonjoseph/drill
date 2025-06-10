package breakpoints

import (
	"strings"

	"github.com/andersonjoseph/drill/internal/components"
	"github.com/andersonjoseph/drill/internal/messages"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type conditionInputModel struct {
	id        int
	isFocused bool
	textInput textinput.Model
	width     int
}

func newConditionInputModel(id int) conditionInputModel {
	ti := textinput.New()
	ti.Placeholder = "condition"
	ti.Width = 0

	return conditionInputModel{
		id:        id,
		textInput: ti,
	}
}

func (m conditionInputModel) Init() tea.Cmd {
	return nil
}

func (m conditionInputModel) Update(msg tea.Msg) (conditionInputModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		var cmd tea.Cmd
		if msg.String() == "esc" {
			m.setFocus(false)
			m.textInput.SetValue("")
			return m, func() tea.Msg {
				return messages.WindowFocused(m.id)
			}
		}

		if msg.String() == "enter" {
			m.setFocus(false)
			content := m.textInput.Value()
			m.textInput.SetValue("")
			return m, tea.Batch(
				func() tea.Msg {
					return messages.WindowFocused(m.id)
				},
				func() tea.Msg {
					return messageNewCondition(content)
				},
			)
		}
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.textInput.Width = m.width - 3
		return m, nil

	case messageClearContent:
		m.textInput.SetValue("")
		return m, nil

	case messageNewContent:
		m.textInput.SetValue(string(msg))
		return m, nil
	}

	return m, nil
}

func (m conditionInputModel) View() string {
	title := listFocusedStyle.Render("Breakpoint Condition")
	titleWidth := lipgloss.Width(title)

	topBorder := listFocusedStyle.Render("┌") + title + listFocusedStyle.Render(strings.Repeat("─", max(m.width-titleWidth, 1))) + listFocusedStyle.Render("┐")
	return lipgloss.JoinVertical(lipgloss.Top,
		topBorder,
		listFocusedStyle.
			Border(lipgloss.NormalBorder()).
			BorderTop(false).
			BorderForeground(listFocusedStyle.GetForeground()).
			Foreground(components.ColorWhite).
			Render(m.textInput.View()),
	)
}

func (m *conditionInputModel) setFocus(f bool) {
	m.isFocused = f
	m.textInput.Focus()
}
