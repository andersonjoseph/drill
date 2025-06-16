package breakpoints

import (
	"github.com/andersonjoseph/drill/internal/components"
	"github.com/andersonjoseph/drill/internal/messages"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var aliasInputStyle lipgloss.Style = lipgloss.NewStyle().
	Foreground(components.ColorWhite).
	Border(lipgloss.NormalBorder()).
	BorderForeground(components.ColorYellow)

type messageNewAlias string

type aliasInputModel struct {
	id        int
	isFocused bool
	textInput textinput.Model
	width     int
}

func newAliasInputModel(id int) aliasInputModel {
	ti := textinput.New()
	ti.Placeholder = "alias"

	return aliasInputModel{
		id:        id,
		textInput: ti,
	}
}

func (m aliasInputModel) Init() tea.Cmd {
	return nil
}

func (m aliasInputModel) Update(msg tea.Msg) (aliasInputModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		var cmd tea.Cmd
		if msg.String() == "esc" {
			m.setFocus(false)
			m.textInput.SetValue("")

			return m, tea.Batch(
				func() tea.Msg {
					return messages.TextInputFocused(false)
				},
				func() tea.Msg {
					return messages.WindowFocused(m.id)
				},
			)
		}

		if msg.String() == "enter" {
			m.setFocus(false)
			content := m.textInput.Value()
			m.textInput.SetValue("")
			return m, tea.Batch(
				func() tea.Msg {
					return messages.TextInputFocused(false)
				},
				func() tea.Msg {
					return messages.WindowFocused(m.id)
				},
				func() tea.Msg {
					return messageNewAlias(content)
				},
			)
		}
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.textInput.Width = m.width - 3
		return m, nil
	}

	return m, nil
}

func (m aliasInputModel) View() string {
	return aliasInputStyle.Render(m.textInput.View())
}

func (m *aliasInputModel) setFocus(f bool) {
	m.isFocused = f
	m.textInput.Focus()
}

func (m *aliasInputModel) setContent(c string) {
	m.textInput.SetValue(c)
}
