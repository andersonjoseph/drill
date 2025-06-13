package localvariables

import (
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/v2/quick"
	"github.com/andersonjoseph/drill/internal/debugger"
	"github.com/andersonjoseph/drill/internal/messages"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/reflow/wordwrap"
)

type messageNewContent debugger.Variable

type VariableViewerModel struct {
	id        int
	isOpen    bool
	isFocused bool
	viewport  viewport.Model
	variable  debugger.Variable
	content   string
}

func newVariableViewer(id int) VariableViewerModel {
	return VariableViewerModel{
		id:       id,
		isOpen:   false,
		viewport: viewport.New(0, 0),
	}
}

func (m VariableViewerModel) Init() tea.Cmd {
	return nil
}

func (m VariableViewerModel) Update(msg tea.Msg) (VariableViewerModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.Height = msg.Height
		m.viewport.Width = msg.Width
		m.viewport.SetContent(wordwrap.String(m.content, m.viewport.Width))
		return m, nil

	case tea.KeyMsg:
		if !m.isOpen {
			return m, nil
		}

		if msg.String() == "esc" {
			m.isOpen = false
			return m, func() tea.Msg {
				return messages.WindowFocused(1)
			}
		}

		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}
func (m VariableViewerModel) View() string {
	return m.viewport.View()
}

func (m *VariableViewerModel) setContent(v debugger.Variable) error {
	m.variable = v

	colorizedContent, err := colorize(v.MultilineValue)
	if err != nil {
		return fmt.Errorf("error setting content: %w", err)
	}
	m.content = colorizedContent

	m.viewport.SetContent(wordwrap.String(m.content, m.viewport.Width))
	return nil
}

func (m *VariableViewerModel) setIsOpen(v bool) {
	m.isOpen = v
}

func (m *VariableViewerModel) setFocus(v bool) {
	m.isFocused = v
}

func colorize(content string) (string, error) {
	sb := strings.Builder{}

	err := quick.Highlight(&sb, content, "go", "terminal8", "gruvbox")
	if err != nil {
		return "", fmt.Errorf("error highlighting the source code: %w", err)
	}

	return sb.String(), nil
}
