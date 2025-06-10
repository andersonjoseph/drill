package sourcecode

import (
	"fmt"
	"strings"

	"github.com/andersonjoseph/drill/internal/components"
	"github.com/andersonjoseph/drill/internal/messages"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	cursorFocusedStyle = lipgloss.NewStyle().Background((components.ColorPurple)).Foreground(components.ColorWhite).Bold(true)
	cursorDefaultStyle = lipgloss.NewStyle().Background(components.ColorBlack)

	lineNumberFocusedStyle = lipgloss.NewStyle().Foreground(components.ColorGrey)
)

type viewportWithCursorModel struct {
	isFocused bool
	width     int
	height    int
	cursor    int
	viewport  viewport.Model
	content   []string
}

func newViewportWithCursor() viewportWithCursorModel {
	return viewportWithCursorModel{
		viewport: viewport.New(0, 0),
	}
}

func (m viewportWithCursorModel) Init() tea.Cmd {
	return nil
}

func (m viewportWithCursorModel) Update(msg tea.Msg) (viewportWithCursorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.RefreshContent:
		m.updateContent()
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.viewport.Height = m.height
		m.viewport.Width = m.width

		m.updateContent()
		return m, nil

	case tea.KeyMsg:

		switch msg.String() {
		case "j", "down":
			m.moveCursor(1)
		case "k", "up":
			m.moveCursor(-1)
		case "g":
			m.cursor = 0
			m.viewport.GotoTop()
			m.updateContent()
		case "G":
			m.cursor = len(m.content) - 1
			m.viewport.GotoBottom()
			m.updateContent()

		default:
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m viewportWithCursorModel) View() string {
	return m.viewport.View()
}

func (m *viewportWithCursorModel) setFocus(f bool) {
	m.isFocused = f
	m.updateContent()
}

func (m viewportWithCursorModel) CurrentLine() string {
	return m.Line(m.cursor)
}

func (m viewportWithCursorModel) CurrentLineNumber() int {
	return m.cursor + 1
}

func (m *viewportWithCursorModel) moveCursor(delta int) {
	newPos := m.cursor + delta
	if newPos >= 0 && newPos < len(m.content) {
		m.cursor = newPos
		m.ensureCursorVisible()
		m.updateContent()
	}
}

func (m *viewportWithCursorModel) jumpToLine(index int) {
	if len(m.content) == 0 {
		return
	}

	if index < 0 {
		index = 0
	} else if index >= len(m.content) {
		index = len(m.content) - 1
	}

	m.cursor = index - 1
	m.ensureCursorVisible()
	m.updateContent()
}

func (m *viewportWithCursorModel) updateContent() {
	if len(m.content) == 0 {
		m.viewport.SetContent("")
		return
	}

	var cursorStyle lipgloss.Style
	if m.isFocused {
		cursorStyle = cursorFocusedStyle
	} else {
		cursorStyle = cursorDefaultStyle
	}

	var lines []string
	for i, line := range m.content {
		lineNumber := lineNumberFocusedStyle.Render(fmt.Sprintf("%4d │ ", i+1))
		if m.cursor == i {
			lineNumber = cursorStyle.Render(lineNumber)
		}

		// Reset ANSI codes at the end of each line to prevent bleeding
		cleanLine := line + "\033[0m"
		lines = append(lines, lineNumber+cleanLine)
	}

	m.viewport.SetContent(strings.Join(lines, "\n"))
}

func (m *viewportWithCursorModel) ensureCursorVisible() {
	top := m.viewport.YOffset
	bottom := top + m.viewport.Height - 1

	if m.cursor < top {
		m.viewport.SetYOffset(m.cursor)
	} else if m.cursor > bottom {
		m.viewport.SetYOffset(m.cursor - m.viewport.Height + 1)
	}
}

func (m viewportWithCursorModel) Line(index int) string {
	if index >= 0 && index < len(m.content) {
		return m.content[index]
	}
	return ""
}

func (m *viewportWithCursorModel) setContent(content []string) {
	m.content = content
	if m.cursor >= len(content) && len(content) > 0 {
		m.cursor = len(content) - 1
	}
	if len(content) == 0 {
		m.cursor = 0
	}
	m.viewport.Width = m.width
	m.viewport.Height = m.height
	m.updateContent()
}
