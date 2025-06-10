package callstack

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/andersonjoseph/drill/internal/components"
	"github.com/andersonjoseph/drill/internal/debugger"
	"github.com/andersonjoseph/drill/internal/messages"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/paginator"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	noItemsStyle lipgloss.Style = lipgloss.NewStyle().Width(0).Foreground(components.ColorGrey)

	paginatorStyleFocused lipgloss.Style = lipgloss.NewStyle().Foreground(components.ColorGreen).PaddingRight(2)
	paginatorStyleDefault lipgloss.Style = lipgloss.NewStyle().Foreground(components.ColorWhite).PaddingRight(2)

	frameStyleTop     lipgloss.Style = lipgloss.NewStyle().Foreground(components.ColorGreen)
	frameStyleFocused lipgloss.Style = lipgloss.NewStyle().Foreground(components.ColorPurple).Bold(true)
	frameStyleDefault lipgloss.Style = lipgloss.NewStyle().Foreground(components.ColorGrey)

	listFocusedStyle lipgloss.Style = lipgloss.NewStyle().Foreground(components.ColorGreen)
	listDefaultStyle lipgloss.Style = lipgloss.NewStyle()
)

type Model struct {
	ID        int
	title     string
	IsFocused bool
	width     int
	height    int
	list      list.Model
	debugger  *debugger.Debugger
}

func New(id int, debugger *debugger.Debugger) Model {
	l := list.New([]list.Item{}, listDelegate{}, 0, 0)
	l.SetShowHelp(false)
	l.SetShowFilter(false)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.Styles.PaginationStyle = paginatorStyleDefault
	l.Styles.NoItems = lipgloss.NewStyle().Width(0)

	p := paginator.New()
	p.Type = paginator.Arabic
	p.PerPage = 5
	p.SetTotalPages(0)
	p.ArabicFormat = lipgloss.NewStyle().
		Margin(0).Padding(0).
		Align(lipgloss.Right).
		Render("%d of %d ")

	l.Paginator = p

	return Model{
		ID:       id,
		title:    "Call Stack",
		list:     l,
		debugger: debugger,
	}
}

func (m Model) Init() tea.Cmd { return nil }
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {

	switch msg := msg.(type) {
	case messages.IsFocused:
		m.IsFocused = bool(msg)
		m.list.SetDelegate(listDelegate{parentFocused: m.IsFocused})

		if !m.IsFocused {
			m.list.Styles.PaginationStyle = paginatorStyleDefault
		} else {
			m.list.Styles.PaginationStyle = paginatorStyleFocused
		}

		return m, nil

	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
		m.list.SetHeight(msg.Height)
		m.list.SetWidth(msg.Width)
		m.list.Styles.NoItems = noItemsStyle.Width(msg.Width)

		return m, nil

	case messages.UpdateContent, messages.Restart:
		if err := m.updateContent(); err != nil {
			return m, func() tea.Msg {
				return messages.Error(err)
			}
		}

		return m, nil

	case tea.KeyMsg:
		var cmd tea.Cmd
		if !m.IsFocused {
			return m, nil
		}

		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) View() string {
	var style lipgloss.Style
	if m.IsFocused {
		style = listFocusedStyle
	} else {
		style = listDefaultStyle
	}

	title := style.Render(fmt.Sprintf("[%d] %s", m.ID, m.title))
	titleWidth := lipgloss.Width(title)
	topBorder := style.Render("┌") + title + style.Render(strings.Repeat("─", max(m.width-titleWidth, 1))) + style.Render("┐")

	return lipgloss.JoinVertical(lipgloss.Top,
		topBorder,
		style.
			Border(lipgloss.NormalBorder()).
			BorderForeground(style.GetForeground()).
			BorderTop(false).
			Render(m.list.View()),
	)
}

func (m *Model) updateContent() error {
	stack, err := m.debugger.CallStack()
	if err != nil {
		return fmt.Errorf("erorr updating content: %w", err)
	}

	m.list.SetItems(stackToListItems(stack))
	return nil
}

func truncPath(path string, maxWidth int) string {
	if len(path) <= maxWidth {
		return path
	}

	dir, filename := filepath.Split(path)
	if len(filename) >= maxWidth {
		return filename
	}

	availableSpace := maxWidth - len(filename) - 3
	if availableSpace <= 0 {
		return filename
	}

	dirParts := strings.Split(strings.TrimSuffix(dir, string(filepath.Separator)), string(filepath.Separator))

	var truncatedDir string

	for i := len(dirParts) - 1; i >= 0; i-- {
		nextPart := dirParts[i]
		if i < len(dirParts)-1 {
			nextPart += string(filepath.Separator)
		}

		if len(nextPart)+len(truncatedDir) > availableSpace {
			if truncatedDir != "" {
				break
			}
			if len(nextPart) > availableSpace {
				truncatedDir = nextPart[len(nextPart)-availableSpace:]
			} else {
				truncatedDir = nextPart
			}
			break
		}

		truncatedDir = nextPart + truncatedDir
	}

	if truncatedDir != "" && !strings.HasSuffix(truncatedDir, string(filepath.Separator)) {
		truncatedDir += string(filepath.Separator)
	}

	return "..." + truncatedDir + filename
}

type listDelegate struct {
	parentFocused bool
}

func (d listDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	listItem := item.(listItem)

	listItem.isFocused = m.Index() == index && d.parentFocused
	fmt.Fprint(w, listItem.Render(m.Width()))
}

func (d listDelegate) Height() int                               { return 1 }
func (d listDelegate) Spacing() int                              { return 0 }
func (d listDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

type listItem struct {
	frame     debugger.StackFrame
	isFocused bool
}

func (i listItem) FilterValue() string { return "" }

func (i listItem) Render(width int) string {
	var style lipgloss.Style
	var indicator string

	truncatedFilename := truncPath(i.frame.Filename, width-2)

	item := fmt.Sprintf("%s()\n %s:%d", i.frame.FunctionName, truncatedFilename, i.frame.Line)

	if i.isFocused {
		style = frameStyleFocused
	} else if i.frame.Index == 0 {
		style = frameStyleTop
	} else {
		style = frameStyleDefault
	}

	stackFrame :=
		lipgloss.JoinHorizontal(lipgloss.Top, indicator, style.Render(item))

	return lipgloss.NewStyle().
		Width(width).
		Render(stackFrame)
}

func stackToListItems(stack []debugger.StackFrame) []list.Item {
	items := make([]list.Item, len(stack))

	for i := range stack {
		items[i] = listItem{
			frame: stack[i],
		}
	}

	return items
}
