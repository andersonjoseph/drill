package sourcecode

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
	cursorFocusedStyle = lipgloss.NewStyle().Background((components.ColorPurple)).Foreground(components.ColorWhite).Bold(true)
	cursorDefaultStyle = lipgloss.NewStyle().Background(components.ColorBlack)

	lineNumberStyle = lipgloss.NewStyle().Foreground(components.ColorGrey)
)

type viewportWithCursorModel struct {
	isFocused bool
	width     int
	height    int
	cursor    int
	viewport  viewport.Model
	content   []string
	filename  string
	debugger  *debugger.Debugger
}

func newViewportWithCursor(debugger *debugger.Debugger) viewportWithCursorModel {
	return viewportWithCursorModel{
		viewport: viewport.New(0, 0),
		debugger: debugger,
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
	m.centerCursorView()
	m.updateContent()
}

func (m *viewportWithCursorModel) centerCursorView() {
	newYOffset := m.cursor - (m.viewport.Height / 2)

	if newYOffset < 0 {
		newYOffset = 0
	}

	maxOffset := len(m.content) - m.viewport.Height
	if maxOffset < 0 {
		maxOffset = 0
	}
	if newYOffset > maxOffset {
		newYOffset = maxOffset
	}

	m.viewport.SetYOffset(newYOffset)
}

func (m *viewportWithCursorModel) updateContent() error {
	if len(m.content) == 0 {
		m.viewport.SetContent("")
		return nil
	}

	arrowLineNumber, err := m.debugger.CurrentLine()
	if err != nil {
		return fmt.Errorf("error updating content: could not get current line: %w", err)
	}

	breakpointsInFile, err := m.debugger.FileBreakpoints(m.filename)
	if err != nil {
		return fmt.Errorf("error updating content: could not get breakpoints: %w", err)
	}

	var cursorStyle lipgloss.Style
	if m.isFocused {
		cursorStyle = cursorFocusedStyle
	} else {
		cursorStyle = cursorDefaultStyle
	}

	lines := make([]string, len(m.content))
	for i, line := range m.content {
		lineNumber := i + 1
		rederedLineNumber := lineNumberStyle.Render(fmt.Sprintf("%4d │ ", lineNumber))
		if m.cursor == i {
			rederedLineNumber = cursorStyle.Render(rederedLineNumber)
		}

		bp, isBpInLine := breakpointsInFile[lineNumber]
		var prefix string
		if lineNumber == arrowLineNumber {
			if isBpInLine && !bp.Disabled {
				prefix = arrowInBreakpoint
			} else {
				prefix = arrow
			}
		} else if isBpInLine {
			if bp.Disabled {
				prefix = disabledBreakpointDot
			} else {
				prefix = enabledBreakpointDot
			}
		} else {
			prefix = "   "
		}
		lines[i] = rederedLineNumber + prefix + line
	}

	m.viewport.SetContent(strings.Join(lines, "\n"))
	return nil
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

func (m *viewportWithCursorModel) setContent(filename string, content []string) {
	m.content = content
	m.filename = filename

	if m.cursor >= len(content) && len(content) > 0 {
		m.cursor = len(content) - 1
	}
	if len(content) == 0 {
		m.cursor = 0
	}
	m.updateContent()
}
