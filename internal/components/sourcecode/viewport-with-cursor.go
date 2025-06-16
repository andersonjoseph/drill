package sourcecode

import (
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/chroma/v2/quick"
	"github.com/andersonjoseph/drill/internal/components"
	"github.com/andersonjoseph/drill/internal/debugger"
	"github.com/andersonjoseph/drill/internal/messages"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hashicorp/golang-lru/v2"
)

const (
	arrowSymbol         = " 🢂 "
	breakpointDotSymbol = " ⏺ "
)

var (
	cursorStyle = lipgloss.NewStyle().Background((components.ColorPurple)).Foreground(components.ColorWhite).Bold(true)

	lineNumberStyle = lipgloss.NewStyle().Foreground(components.ColorGrey)

	enabledBreakpointDot  = lipgloss.NewStyle().Foreground(components.ColorRed).Render(breakpointDotSymbol)
	disabledBreakpointDot = lipgloss.NewStyle().Foreground(components.ColorGrey).Render(breakpointDotSymbol)

	arrow             = lipgloss.NewStyle().Foreground(components.ColorGreen).Render(arrowSymbol)
	arrowInBreakpoint = lipgloss.NewStyle().Foreground(components.ColorRed).Render(arrowSymbol)
)

type viewportWithCursorModel struct {
	width            int
	height           int
	cursor           int
	arrowLineNumber  int
	filename         string
	content          []string
	formattedContent []string
	contentCache     *lru.Cache[string, []string]
	breakpoints      map[int]debugger.Breakpoint
	debugger         *debugger.Debugger
	viewport         viewport.Model
}

func newViewportWithCursor(debugger *debugger.Debugger) viewportWithCursorModel {
	contentCache, err := lru.New[string, []string](5)
	if err != nil {
		panic(err)
	}
	return viewportWithCursorModel{
		cursor:       1,
		viewport:     viewport.New(0, 0),
		debugger:     debugger,
		contentCache: contentCache,
	}
}

func (m viewportWithCursorModel) Init() tea.Cmd {
	return nil
}

func (m viewportWithCursorModel) Update(msg tea.Msg) (viewportWithCursorModel, tea.Cmd) {
	switch msg := msg.(type) {

	case messages.DebuggerStepped:
		filename, line, err := m.debugger.CurrentFile()
		if err != nil {
			return m, messages.ErrorCmd(fmt.Errorf("error refreshing content: could not get current file: %w", err))
		}

		if m.filename != filename {
			if err := m.openFile(filename, line); err != nil {
				return m, messages.ErrorCmd(fmt.Errorf("error refreshing content: could not get current file: %w", err))
			}
			m.centerCursor()
		}

		oldArrowLineNumber := m.arrowLineNumber
		m.arrowLineNumber = line

		m.renderLine(oldArrowLineNumber)
		m.renderLine(m.arrowLineNumber)

		m.setCursor(m.arrowLineNumber)
		m.centerCursor()
		return m, nil

	case messages.RefreshContent, messages.DebuggerRestarted:
		filename, line, err := m.debugger.CurrentFile()
		if err != nil {
			return m, messages.ErrorCmd(fmt.Errorf("error refreshing content: could not get current file: %w", err))
		}

		if err := m.openFile(filename, line); err != nil {
			return m, messages.ErrorCmd(fmt.Errorf("error refreshing content: could not get current file: %w", err))
		}

		m.centerCursor()
		return m, nil

	case messages.FileRequested:
		if err := m.openFile(msg.Filename, msg.Line); err != nil {
			return m, messages.ErrorCmd(fmt.Errorf("error handling file requested: %w", err))
		}
		m.centerCursor()
		return m, nil

	case messages.DebuggerBreakpointCreated:
		if msg.Filename != m.filename {
			return m, nil
		}

		bp, err := m.debugger.Breakpoint(msg.ID)
		if err != nil {
			return m, messages.ErrorCmd(fmt.Errorf("could not get breakpoints: %w", err))
		}

		m.breakpoints[msg.Line] = bp
		m.renderLine(msg.Line)
		return m, nil

	case messages.DebuggerBreakpointToggled:
		if msg.Filename != m.filename {
			return m, nil
		}

		if bp, ok := m.breakpoints[msg.Line]; ok {
			bp.Disabled = !bp.Disabled
			m.breakpoints[msg.Line] = bp
		}

		m.renderLine(msg.Line)
		return m, nil

	case messages.DebuggerBreakpointCleared:
		if msg.Filename != m.filename {
			return m, nil
		}

		delete(m.breakpoints, msg.Line)
		m.renderLine(msg.Line)
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.viewport.Height = m.height
		m.viewport.Width = m.width
		m.centerCursor()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			m.setCursor(m.cursor + 1)
			return m, nil
		case "k", "up":
			m.setCursor(m.cursor - 1)
			return m, nil
		case "g":
			m.setCursor(1)
			m.viewport.GotoTop()
			return m, nil
		case "G":
			m.setCursor(len(m.content))
			m.viewport.GotoBottom()
			return m, nil

		case "z":
			m.centerCursor()
			return m, nil

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

func (m *viewportWithCursorModel) setCursor(newPos int) {
	if newPos < 1 || newPos > len(m.content) {
		return
	}

	oldPos := m.cursor
	if oldPos == newPos {
		return
	}

	m.cursor = newPos

	m.renderLine(oldPos)
	m.renderLine(newPos)

	m.viewport.SetContent(strings.Join(m.formattedContent, "\n"))
	m.ensureCursorVisible()
}

func (m *viewportWithCursorModel) renderLine(line int) error {
	newLine, err := m.formatLine(line)
	if err != nil {
		return err
	}

	m.formattedContent[line-1] = newLine
	m.viewport.SetContent(strings.Join(m.formattedContent, "\n"))
	m.ensureCursorVisible()

	return nil
}

func (m *viewportWithCursorModel) centerCursor() {
	maxOffset := max(0, len(m.content)-m.viewport.Height)

	newYOffset := m.cursor - 1 - (m.viewport.Height / 2)

	clampedOffset := max(0, min(newYOffset, maxOffset))

	m.viewport.SetYOffset(clampedOffset)
}

func (m *viewportWithCursorModel) formatLine(lineNumber int) (string, error) {
	lineIndex := lineNumber - 1
	lineContent := m.content[lineIndex]

	var prefix string
	bp, isBpInLine := m.breakpoints[lineNumber]

	rederedLineNumber := lineNumberStyle.Render(fmt.Sprintf("%4d │ ", lineNumber))
	if m.cursor == lineNumber {
		rederedLineNumber = cursorStyle.Render(rederedLineNumber)
	}

	if lineNumber == m.arrowLineNumber {
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

	return rederedLineNumber + prefix + lineContent, nil
}

func (m *viewportWithCursorModel) ensureCursorVisible() {
	cursorIdx := m.cursor - 1

	top := m.viewport.YOffset
	bottom := top + m.viewport.Height - 1

	if cursorIdx < top {
		m.viewport.SetYOffset(cursorIdx)
	} else if cursorIdx > bottom {
		newYOffset := cursorIdx - m.viewport.Height + 1
		m.viewport.SetYOffset(newYOffset)
	}
}

func (m *viewportWithCursorModel) openFile(filename string, line int) error {
	m.filename = filename
	m.cursor = line
	if content, ok := m.contentCache.Get(filename); ok {
		m.content = content
	} else {
		content, err := m.readFile(filename)
		if err != nil {
			return fmt.Errorf("error opening file: could not read file: %w", err)
		}

		content, err = colorize(content)
		if err != nil {
			return fmt.Errorf("error opening file: could not colorize content: %w", err)
		}
		m.content = strings.Split(strings.TrimSpace(content), "\n")
		m.contentCache.Add(filename, m.content)
	}

	breakpointsInFile, err := m.debugger.FileBreakpoints(m.filename)
	if err != nil {
		return fmt.Errorf("could not get breakpoints: %w", err)
	}
	m.breakpoints = breakpointsInFile

	_, arrowLineNumber, err := m.debugger.CurrentFile()
	if err != nil {
		return fmt.Errorf("could not get current line: %w", err)
	}
	m.arrowLineNumber = arrowLineNumber

	m.formattedContent = make([]string, len(m.content))
	for i := range m.content {
		renderedLine, err := m.formatLine(i + 1)
		if err != nil {
			return err
		}
		m.formattedContent[i] = renderedLine
	}

	m.viewport.SetContent(strings.Join(m.formattedContent, "\n"))
	return nil
}

func (m *viewportWithCursorModel) readFile(filename string) (string, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("error getting current file content: error opening file: %s: %v", filename, err)
	}

	return string(content), nil
}

func colorize(content string) (string, error) {
	sb := strings.Builder{}

	err := quick.Highlight(&sb, content, "go", "terminal8", "native")
	if err != nil {
		return "", fmt.Errorf("error highlighting the source code: %w", err)
	}

	return sb.String(), nil
}

func (m viewportWithCursorModel) CurrentLineNumber() int {
	return m.cursor
}
