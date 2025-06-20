package breakpoints

import (
	"cmp"
	"fmt"
	"io"
	"slices"

	"github.com/andersonjoseph/drill/internal/components"
	"github.com/andersonjoseph/drill/internal/debugger"
	"github.com/andersonjoseph/drill/internal/messages"
	"github.com/andersonjoseph/drill/internal/paths"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/paginator"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	breakpointSymbol = "⏺"
	hintString       = "t: toggle, d: delete, enter: select, c: condition, r: alias, j: down, k: up"
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
	ID              int
	title           string
	IsFocused       bool
	width           int
	height          int
	list            list.Model
	debugger        *debugger.Debugger
	conditionInput  conditionInputModel
	aliasInput      aliasInputModel
	idToBreakpoints map[int]debugger.Breakpoint
}

func New(id int, d *debugger.Debugger) Model {
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
		ID:              id,
		title:           "Breakpoints",
		list:            l,
		debugger:        d,
		conditionInput:  newConditionInputModel(id),
		aliasInput:      newAliasInputModel(id),
		idToBreakpoints: make(map[int]debugger.Breakpoint),
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
			return m, nil
		} else {
			m.list.Styles.PaginationStyle = paginatorStyleFocused
		}

		return m, func() tea.Msg {
			return messages.UpdatedHint(hintString)
		}

	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
		m.list.SetHeight(msg.Height)
		m.list.SetWidth(msg.Width)
		m.list.Styles.NoItems = noItemsStyle.Width(msg.Width)

		m.conditionInput, _ = m.conditionInput.Update(tea.WindowSizeMsg{Width: m.width})
		m.aliasInput, _ = m.aliasInput.Update(tea.WindowSizeMsg{Width: m.width})
		return m, nil

	case messages.RefreshContent, messages.DebuggerRestarted:
		if err := m.syncBreakpoints(); err != nil {
			return m, messages.ErrorCmd(err)
		}
		return m, nil

	case messages.DebuggerBreakpointCreated:
		bp, err := m.debugger.Breakpoint(msg.ID)
		if err != nil {
			return m, messages.ErrorCmd(fmt.Errorf("error getting breakpoint: %w", err))
		}
		m.idToBreakpoints[msg.ID] = bp

		m.list.SetItems(breakpointsToListItems(m.idToBreakpoints))
		return m, nil

	case messages.DebuggerBreakpointToggled:
		bp := m.idToBreakpoints[msg.ID]

		bp.Disabled = !bp.Disabled
		m.idToBreakpoints[msg.ID] = bp

		m.list.SetItems(breakpointsToListItems(m.idToBreakpoints))
		return m, nil

	case messages.DebuggerBreakpointCleared:
		delete(m.idToBreakpoints, msg.ID)

		m.list.SetItems(breakpointsToListItems(m.idToBreakpoints))
		return m, nil

	case messageNewCondition:
		item := m.list.SelectedItem().(listItem)

		bp, err := m.debugger.AddConditionToBreakpoint(item.breakpoint.ID, string(msg))
		if err != nil {
			return m, messages.ErrorCmd(err)
		}

		m.idToBreakpoints[bp.ID] = bp

		m.list.SetItems(breakpointsToListItems(m.idToBreakpoints))
		return m, nil

	case messageNewAlias:
		item := m.list.SelectedItem().(listItem)

		bp, err := m.debugger.AddAliasToBreakpoint(item.breakpoint.ID, string(msg))
		if err != nil {
			return m, messages.ErrorCmd(err)
		}

		m.idToBreakpoints[bp.ID] = bp
		m.list.SetItems(breakpointsToListItems(m.idToBreakpoints))
		return m, nil

	case messages.DebuggerBreakpointSelected:
		if msg.FromWindowID == m.ID {
			return m, nil
		}

		for i, item := range m.list.Items() {
			item := item.(listItem)
			if item.breakpoint.ID == int(msg.ID) {
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

		if m.aliasInput.isFocused {
			m.aliasInput, cmd = m.aliasInput.Update(msg)
			return m, cmd
		}

		if !m.IsFocused {
			return m, nil
		}

		if msg.String() == "t" {
			bp, err := m.toggleBreakpoint()
			if err != nil {
				return m, messages.ErrorCmd(err)
			}

			return m, messages.DebuggerBreakpointToggledCmd(bp.ID, bp.Filename, bp.Line)
		}

		if msg.String() == "d" {
			bp, err := m.clearBreakpoint()
			if err != nil {
				return m, messages.ErrorCmd(err)
			}

			return m, messages.DebuggerBreakpointClearedCmd(bp.ID, bp.Filename, bp.Line)
		}

		if msg.String() == "c" {
			if m.list.SelectedItem() == nil {
				return m, nil
			}
			item := m.list.SelectedItem().(listItem)

			m.conditionInput.setFocus(true)
			m.conditionInput.setContent(item.breakpoint.Condition)
			return m, func() tea.Msg {
				return messages.TextInputFocused(true)
			}
		}

		if msg.String() == "r" {
			if m.list.SelectedItem() == nil {
				return m, nil
			}

			item := m.list.SelectedItem().(listItem)

			m.aliasInput.setFocus(true)
			m.aliasInput.setContent(item.breakpoint.Name)
			return m, func() tea.Msg {
				return messages.TextInputFocused(true)
			}
		}

		if msg.String() == "enter" {
			if m.list.SelectedItem() == nil {
				return m, nil
			}

			item := m.list.SelectedItem().(listItem)
			return m, messages.DebuggerBreakpointSelectedCmd(
				item.breakpoint.ID,
				item.breakpoint.Filename,
				item.breakpoint.Line,
				m.ID,
			)
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
	if m.aliasInput.isFocused {
		return m.aliasInput.View()
	}

	return m.list.View()
}

func (m *Model) syncBreakpoints() error {
	bps, err := m.debugger.Breakpoints()
	if err != nil {
		return fmt.Errorf("erorr updating content: %w", err)
	}

	m.idToBreakpoints = make(map[int]debugger.Breakpoint, len(bps))
	for _, bp := range bps {
		if _, ok := m.idToBreakpoints[bp.ID]; !ok {
			m.idToBreakpoints[bp.ID] = bp
		}
	}

	m.list.SetItems(breakpointsToListItems(m.idToBreakpoints))
	return nil
}

func (m *Model) toggleBreakpoint() (debugger.Breakpoint, error) {
	i := m.list.SelectedItem()
	if i == nil {
		return debugger.Breakpoint{}, nil
	}
	bp := i.(listItem).breakpoint
	err := m.debugger.ToggleBreakpoint(bp.ID)
	if err != nil {
		return debugger.Breakpoint{}, err
	}
	return bp, nil
}

func (m *Model) clearBreakpoint() (debugger.Breakpoint, error) {
	i := m.list.SelectedItem()
	if i == nil {
		return debugger.Breakpoint{}, nil
	}
	bp := i.(listItem).breakpoint
	err := m.debugger.ClearBreakpoint(bp.ID)
	if err != nil {
		return debugger.Breakpoint{}, err
	}

	return bp, nil
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

	var item string
	if i.breakpoint.Condition != "" {
		item = conditionStyle.Render("when", i.breakpoint.Condition, "")
	}

	var style lipgloss.Style
	if i.isFocused {
		style = breakpointStyleFocused
	} else {
		style = breakpointStyleDefault
	}

	item += style.Render(paths.Trunc(i.breakpoint.Name, width-len(item)-5))

	breakpoint :=
		lipgloss.JoinHorizontal(lipgloss.Top, indicator, item)

	return listItemStyle.
		Width(width).
		Render(breakpoint)
}

func breakpointsToListItems(bps map[int]debugger.Breakpoint) []list.Item {
	items := make([]list.Item, 0, len(bps))
	for _, bp := range bps {
		items = append(items, listItem{breakpoint: bp})
	}

	slices.SortFunc(items, func(a, b list.Item) int {
		return cmp.Compare(a.(listItem).breakpoint.ID, b.(listItem).breakpoint.ID)
	})

	return items
}
