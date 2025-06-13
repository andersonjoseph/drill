package breakpoints

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

const (
	breakpointSymbol = "⏺"
)

var (
	noItemsStyle lipgloss.Style = lipgloss.NewStyle().Width(0).Foreground(components.ColorGrey)

	paginatorStyleFocused lipgloss.Style = lipgloss.NewStyle().Foreground(components.ColorGreen).PaddingRight(2)
	paginatorStyleDefault lipgloss.Style = lipgloss.NewStyle().Foreground(components.ColorWhite).PaddingRight(2)

	breakpointStyleFocused lipgloss.Style = lipgloss.NewStyle().Foreground(components.ColorPurple).Bold(true)
	breakpointStyleDefault lipgloss.Style = lipgloss.NewStyle().Foreground(components.ColorGrey)

	indicatorEnabled  = lipgloss.NewStyle().Foreground(components.ColorRed).Render(breakpointSymbol, " ")
	indicatorDisabled = lipgloss.NewStyle().Foreground(components.ColorGrey).Render(breakpointSymbol, " ")
	conditionStyle    = lipgloss.NewStyle().Foreground(components.ColorYellow)

	listItemStyle = lipgloss.NewStyle()
)

type Model struct {
	ID             int
	title          string
	IsFocused      bool
	width          int
	height         int
	list           list.Model
	debugger       *debugger.Debugger
	conditionInput conditionInputModel
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
		ID:             id,
		title:          "Breakpoints",
		list:           l,
		debugger:       debugger,
		conditionInput: newConditionInputModel(id),
	}
}

func (m Model) Init() tea.Cmd { return nil }
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {
	case messages.WindowFocused:
		m.IsFocused = int(msg) == m.ID
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

		m.conditionInput, _ = m.conditionInput.Update(tea.WindowSizeMsg{Width: m.width})
		return m, nil

	case
		messages.RefreshContent,
		messages.DebuggerRestarted,
		messages.DebuggerBreakpointCreated,
		messages.DebuggerBreakpointToggled,
		messages.DebuggerBreakpointCleared:

		if err := m.updateContent(); err != nil {
			return m, func() tea.Msg {
				return messages.Error(err)
			}
		}

		return m, nil

	case messageNewCondition:
		var cmd tea.Cmd
		bp := m.list.SelectedItem().(listItem)
		_, err := m.debugger.AddConditionToBreakpoint(bp.breakpoint.ID, string(msg))

		if err != nil {
			return m, func() tea.Msg {
				return messages.Error(err)
			}
		}

		if err := m.updateContent(); err != nil {
			return m, func() tea.Msg {
				return messages.Error(err)
			}
		}

		return m, cmd

	case messages.BreakpointSelected:
		for i, item := range m.list.Items() {
			item := item.(listItem)
			if item.breakpoint.ID == int(msg) {
				m.list.Select(i)
				return m, func() tea.Msg {
					return messages.WindowFocused(m.ID)
				}
			}
		}

		return m, nil

	case tea.KeyMsg:
		var cmd tea.Cmd
		if m.conditionInput.isFocused {
			m.conditionInput, cmd = m.conditionInput.Update(msg)
			return m, cmd
		}

		if !m.IsFocused {
			return m, nil
		}

		if msg.String() == "t" {
			m.toggleBreakpoint()
			return m, func() tea.Msg {
				return messages.DebuggerBreakpointToggled{}
			}
		}

		if msg.String() == "d" {
			if err := m.clearBreakpoint(); err != nil {
				return m, func() tea.Msg {
					return messages.Error(err)
				}
			}
			return m, func() tea.Msg {
				return messages.DebuggerBreakpointCleared{}
			}
		}

		if msg.String() == "c" {
			if m.list.SelectedItem() == nil {
				return m, nil
			}
			bp := m.list.SelectedItem().(listItem)

			m.conditionInput.setFocus(true)
			m.conditionInput.setContent(bp.breakpoint.Condition)
			return m, func() tea.Msg {
				return messages.TextInputFocused(true)
			}
		}

		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) View() string {
	if m.conditionInput.isFocused {
		return m.conditionInput.View()
	}
	return m.list.View()
}

func (m *Model) updateContent() error {
	bps, err := m.debugger.Breakpoints()
	if err != nil {
		return fmt.Errorf("erorr updating content: %w", err)
	}

	m.list.SetItems(breakpointsToListItems(bps))
	return nil
}

func (m *Model) toggleBreakpoint() {
	i := m.list.SelectedItem()
	if i == nil {
		return
	}
	id := i.(listItem).breakpoint.ID
	m.debugger.ToggleBreakpoint(id)
}

func (m *Model) clearBreakpoint() error {
	i := m.list.SelectedItem()
	if i == nil {
		return nil
	}
	id := i.(listItem).breakpoint.ID
	m.debugger.ClearBreakpoint(id)

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

func (d listDelegate) Height() int                               { return 2 }
func (d listDelegate) Spacing() int                              { return 0 }
func (d listDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

type listItem struct {
	breakpoint debugger.Breakpoint
	isFocused  bool
}

func (i listItem) FilterValue() string { return "" }

func (i listItem) Render(width int) string {
	var indicator string
	if i.breakpoint.Disabled {
		indicator = indicatorDisabled
	} else {
		indicator = indicatorEnabled
	}

	item := truncPath(i.breakpoint.Name, width-3)
	if i.breakpoint.Condition != "" {
		item += conditionStyle.Render("\n\twhen: " + i.breakpoint.Condition)
	}

	var style lipgloss.Style
	if i.isFocused {
		style = breakpointStyleFocused
	} else {
		style = breakpointStyleDefault
	}

	breakpoint :=
		lipgloss.JoinHorizontal(lipgloss.Top, indicator, style.Render(item))

	return listItemStyle.
		Width(width).
		Render(breakpoint)
}

func breakpointsToListItems(bps []debugger.Breakpoint) []list.Item {
	items := make([]list.Item, len(bps))

	for i := range bps {
		items[i] = listItem{
			breakpoint: bps[i],
		}
	}

	return items
}
