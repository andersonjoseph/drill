package output

import (
	"errors"
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/v2/quick"
	"github.com/andersonjoseph/drill/internal/components"
	"github.com/andersonjoseph/drill/internal/debugger"
	"github.com/andersonjoseph/drill/internal/messages"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	promptStyle lipgloss.Style = lipgloss.NewStyle().
			Foreground(components.ColorWhite)
	commandStyle lipgloss.Style = lipgloss.NewStyle().Foreground(components.ColorGrey)
	errorStyle   lipgloss.Style = lipgloss.NewStyle().Foreground(components.ColorRed)
)

type CommandInputModel struct {
	ID        int
	IsFocused bool
	textInput textinput.Model
	debugger  *debugger.Debugger
}

func newCommandInputModel(id int, d *debugger.Debugger) CommandInputModel {
	ti := textinput.New()
	ti.Placeholder = "command..."
	ti.CharLimit = 256
	ti.Width = 80
	ti.Prompt = "> "
	ti.PromptStyle = promptStyle

	return CommandInputModel{
		ID:        id,
		IsFocused: false,
		textInput: ti,
		debugger:  d,
	}
}

func (m CommandInputModel) Init() tea.Cmd {
	return nil
}

func (m CommandInputModel) Update(msg tea.Msg) (CommandInputModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case messages.WindowFocused:
		m.IsFocused = int(msg) == m.ID
		if m.IsFocused {
			m.textInput.Focus()
		} else {
			m.textInput.Blur()
		}

		return m, tea.Batch(
			func() tea.Msg {
				return messages.TextInputFocused(m.IsFocused)
			},
			func() tea.Msg {
				return messages.WindowTitleChanged{
					WindowID: m.ID,
					Title:    "Command Mode",
				}
			},
		)

	case tea.KeyMsg:
		if !m.IsFocused {
			return m, nil
		}

		switch msg.Type {
		case tea.KeyEnter:
			input := strings.TrimSpace(m.textInput.Value())
			if input == "" {
				return m, nil
			}

			parts := strings.Fields(input)
			command, args := parts[0], parts[1:]

			m.textInput.Reset()

			switch command {
			case "print", "p":
				if err := m.commandPrint(input, args); err != nil {
					m.sendOutput(errorStyle.Render(err.Error()))
					return m, nil
				}

			default:
				m.sendOutput(errorStyle.Render(fmt.Sprintf("Error: unrecognized command '%s'", command)))
			}
			return m, nil

		case tea.KeyEsc:
			m.IsFocused = false
			m.textInput.Blur()
			return m, tea.Batch(
				func() tea.Msg {
					return messages.TextInputFocused(false)
				},
				func() tea.Msg {
					return messages.WindowTitleChanged{
						WindowID: m.ID,
						Title:    "Output",
					}
				},
			)
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m CommandInputModel) commandPrint(input string, args []string) error {
	if len(args) == 0 {
		return errors.New("print command requires an argument")
	}

	expr := strings.Join(args, "")
	v, err := m.debugger.EvalVariable(expr)
	if err != nil {
		return err
	}

	colorizedValue, err := Colorize(v.MultilineValue)
	if err != nil {
		colorizedValue = v.MultilineValue
	}

	m.sendOutput(fmt.Sprintf(
		"%s \n%s",
		commandStyle.Render(input),
		colorizedValue,
	))

	return nil
}

func (m CommandInputModel) View() string {
	return m.textInput.View()
}

func (m CommandInputModel) sendOutput(content string) {
	m.debugger.Output <- debugger.Output{
		Source:  debugger.SourceCommand,
		Content: content,
	}
}

func Colorize(content string) (string, error) {
	sb := strings.Builder{}

	err := quick.Highlight(&sb, content, "go", "terminal8", "native")
	if err != nil {
		return "", fmt.Errorf("error highlighting the source code: %w", err)
	}

	return sb.String(), nil
}
