package breakpoints

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/andersonjoseph/drill/internal/debugger"
	"github.com/andersonjoseph/drill/internal/messages"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/paginator"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	colorBlack  = lipgloss.Color("0")
	colorWhite  = lipgloss.Color("15")
	colorGrey   = lipgloss.Color("7")
	colorPurple = lipgloss.Color("5")
	colorGreen  = lipgloss.Color("2")
)

var (
	noItemsStyle lipgloss.Style = lipgloss.NewStyle().Width(0).Foreground(colorGrey)

	paginatorStyleFocused lipgloss.Style = lipgloss.NewStyle().Foreground(colorGreen).PaddingRight(2)
	paginatorStyleDefault lipgloss.Style = lipgloss.NewStyle().Foreground(colorWhite).PaddingRight(2)

	breakpointStyleFocused  lipgloss.Style = lipgloss.NewStyle().Foreground(colorPurple)
	breakpointStyleDisabled lipgloss.Style = lipgloss.NewStyle().Foreground(colorGrey)
	breakpointStyleDefault  lipgloss.Style = lipgloss.NewStyle().Foreground(colorWhite)

	listFocusedStyle lipgloss.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(colorGreen))
	listDefaultStyle lipgloss.Style = lipgloss.NewStyle()
)

type Model struct {
	id        int
	title     string
	isFocused bool
	width     int
	height    int
	list      list.Model
	Error     error
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
	l.Paginator = setupPagination(0)

	return Model{
		id:        id,
		title:     "Breakpoints",
		isFocused: id == 1,
		list:      l,
		debugger:  debugger,
	}
}

func setupPagination(totalItems int) paginator.Model {
	p := paginator.New()
	p.Type = paginator.Arabic
	p.PerPage = 5
	p.SetTotalPages(totalItems)
	p.ArabicFormat = lipgloss.NewStyle().
		Margin(0).Padding(0).
		Align(lipgloss.Right).
		Render("%d of %d ")

	return p
}

func (m Model) Init() tea.Cmd { return nil }
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.handleResize(msg.Height, msg.Width)
		return m, nil

	case messages.UpdateContent:
		m.updateContent()
		return m, nil

	case messages.CreateBreakpointNow:
		if !m.isFocused {
			return m, nil
		}

		m.debugger.CreateBreakpointNow()
		m.updateContent()
		return m, nil

	case messages.ToggleBreakpoint:
		if !m.isFocused {
			return m, nil
		}
		i := m.list.SelectedItem()
		if i == nil {
			return m, nil
		}
		id := i.(listItem).breakpoint.ID
		m.debugger.ToggleBreakpoint(id)
		m.updateContent()
		return m, nil

	case messages.ClearBreakpoint:
		if !m.isFocused {
			return m, nil
		}
		i := m.list.SelectedItem()
		if i == nil {
			return m, nil
		}
		id := i.(listItem).breakpoint.ID
		m.debugger.ClearBreakpoint(id)
		m.updateContent()
		m.list.CursorUp()

		return m, nil

	case tea.KeyMsg:
		if id, err := strconv.Atoi(msg.String()); err == nil {
			m.isFocused = id == m.id
		}
		if !m.isFocused {
			return m, nil
		}

		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	if !m.isFocused {
		m.list.Styles.PaginationStyle = paginatorStyleDefault
		return m, nil
	}

	m.list.Styles.PaginationStyle = paginatorStyleFocused

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)

	return m, cmd
}

func (m Model) View() string {
	var style lipgloss.Style
	if m.isFocused {
		style = listFocusedStyle
	} else {
		style = listDefaultStyle
	}

	width := m.list.Width()
	titleText := style.Render(fmt.Sprintf("%s [%d]", m.title, m.id))
	titleWidth := lipgloss.Width(titleText)

	topBorder := style.Render("┌") + titleText + style.Render(strings.Repeat("─", max(width-titleWidth, 1))) + style.Render("┐")
	bottomBorder := style.Render("└" + strings.Repeat("─", width) + "┘")
	verticalBorder := style.Render("│")

	lines := strings.Split(m.list.View(), "\n")
	renderedLines := []string{topBorder}

	for _, line := range lines {
		paddedLine := verticalBorder + line + verticalBorder
		renderedLines = append(renderedLines, paddedLine)
	}

	renderedLines = append(renderedLines, bottomBorder)
	return strings.Join(renderedLines, "\n")
}

func (m *Model) handleResize(h, w int) {
	m.width = w / 3
	if m.width >= 40 {
		m.width = 40
	} else if m.width <= 20 {
		m.width = 20
	}
	m.list.SetWidth(m.width)
	m.list.Styles.NoItems = noItemsStyle.Width(m.width)

	m.height = h / 3
	if m.height >= 10 {
		m.height = 10
	} else if m.height <= 3 {
		m.height = 3
	}
	m.list.SetHeight(m.height)
}

func (m *Model) updateContent() {
	bps, err := m.debugger.GetBreakpoints()
	if err != nil {
		m.Error = fmt.Errorf("erorr updating content: %w", err)
		return
	}

	m.list.SetItems(breakpointsToListItems(bps))
}

type listDelegate struct{}

func (d listDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	listItem := item.(listItem)

	listItem.isFocused = m.Index() == index
	fmt.Fprint(w, listItem.Render(m.Width()))
}

func (d listDelegate) Height() int                               { return 1 }
func (d listDelegate) Spacing() int                              { return 0 }
func (d listDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

type listItem struct {
	breakpoint debugger.Breakpoint
	isFocused  bool
}

func (i listItem) FilterValue() string { return "" }

func (i listItem) Render(width int) string {
	var style lipgloss.Style

	if i.isFocused {
		style = breakpointStyleFocused
	} else if i.breakpoint.Disabled {
		style = breakpointStyleDisabled
	} else {
		style = breakpointStyleDefault
	}

	breakpoint := style.Render(i.breakpoint.Name)

	return lipgloss.NewStyle().
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
