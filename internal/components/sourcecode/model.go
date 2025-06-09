package sourcecode

import (
	"fmt"
	"strings"

	"github.com/andersonjoseph/drill/internal/components"
	"github.com/andersonjoseph/drill/internal/debugger"
	"github.com/andersonjoseph/drill/internal/messages"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	windowFocusedStyle lipgloss.Style = lipgloss.NewStyle().
				Foreground(components.ColorGreen).
				Border(lipgloss.NormalBorder()).
				BorderTop(false).
				BorderForeground(components.ColorGreen)

	windowDefaultStyle lipgloss.Style = lipgloss.NewStyle().
				Foreground(components.ColorWhite).
				Border(lipgloss.NormalBorder()).BorderTop(false).
				BorderTop(false).
				BorderForeground(components.ColorWhite)
)

type Model struct {
	ID              int
	title           string
	currentFilename string
	IsFocused       bool
	content         string
	Width           int
	Height          int
	debugger        *debugger.Debugger
}

func New(id int, title string, d *debugger.Debugger) Model {
	return Model{
		ID:       id,
		title:    title,
		debugger: d,
	}
}

func (m Model) Init() tea.Cmd { return nil }
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.IsFocused:
		m.IsFocused = bool(msg)
		return m, nil

	case messages.UpdateContent, tea.WindowSizeMsg:
		if err := m.updateContent(); err != nil {
			return m, func() tea.Msg {
				return messages.Error(err)
			}
		}
		return m, nil

	case tea.KeyMsg:
		if !m.IsFocused {
			return m, nil
		}

		if msg.String() == "n" {
			_, err := m.debugger.Client.Next()

			if err != nil {
				err = fmt.Errorf("error stepping over to the next line: %w", err)
				return m, func() tea.Msg {
					return messages.Error(err)
				}
			}

			return m, func() tea.Msg {
				return messages.UpdateContent{}
			}
		}

		if msg.String() == "c" {
			<-m.debugger.Client.Continue()
			return m, func() tea.Msg {
				return messages.UpdateContent{}
			}
		}

		if msg.String() == "r" {
			_, err := m.debugger.Client.Restart(false)
			if err != nil {
				err = fmt.Errorf("error restarting debugger: %w", err)
				return m, func() tea.Msg {
					return messages.Error(err)
				}
			}

			if err := m.updateContent(); err != nil {
				return m, func() tea.Msg {
					return messages.Error(err)
				}
			}

			return m, func() tea.Msg {
				return messages.Restart{}
			}
		}
	}

	return m, nil
}

func (m Model) View() string {
	var style lipgloss.Style
	if m.IsFocused {
		style = windowFocusedStyle
	} else {
		style = windowDefaultStyle
	}

	title := fmt.Sprintf("[%d] %s [%s]", m.ID, m.title, m.currentFilename)

	topBorder := "┌" + title + strings.Repeat("─", max(m.Width-len(title), 1)) + "┐"

	return lipgloss.JoinVertical(
		lipgloss.Top,
		style.Border(lipgloss.Border{}).Render(topBorder),
		style.
			Height(m.Height).
			Width(m.Width).
			Render(m.content),
	)
}

func (m *Model) updateContent() error {
	var err error
	content, err := m.debugger.GetCurrentFileContent((m.Height / 2) - 2)
	if err != nil {
		return fmt.Errorf("error updating content: %w", err)
	}
	m.content = content

	m.currentFilename, err = m.debugger.GetCurrentFilename()
	if err != nil {
		return fmt.Errorf("error getting the current file: %w", err)
	}

	return nil
}
