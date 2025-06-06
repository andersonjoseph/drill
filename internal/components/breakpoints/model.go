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
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	noItemsStyle lipgloss.Style = lipgloss.NewStyle().Width(0).Foreground(components.ColorGrey)

	paginatorStyleFocused lipgloss.Style = lipgloss.NewStyle().Foreground(components.ColorGreen).PaddingRight(2)
	paginatorStyleDefault lipgloss.Style = lipgloss.NewStyle().Foreground(components.ColorWhite).PaddingRight(2)

	breakpointStyleFocused lipgloss.Style = lipgloss.NewStyle().Foreground(components.ColorPurple)
	breakpointStyleDefault lipgloss.Style = lipgloss.NewStyle().Foreground(components.ColorGrey)

	listFocusedStyle lipgloss.Style = lipgloss.NewStyle().Foreground(components.ColorGreen)
	listDefaultStyle lipgloss.Style = lipgloss.NewStyle()
)

type Model struct {
	ID              int
	title           string
	IsFocused       bool
	Width           int
	Height          int
	list            list.Model
	Error           error
	debugger        *debugger.Debugger
	addingCondition bool
	conditionInput  textinput.Model
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

	ti := textinput.New()
	ti.Placeholder = "condition"
	ti.Width = 0

	return Model{
		ID:             id,
		title:          "Breakpoints",
		list:           l,
		debugger:       debugger,
		conditionInput: ti,
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
	case messages.IsFocused:
		m.IsFocused = bool(msg)
		m.list.SetDelegate(listDelegate{parentFocused: m.IsFocused})
		return m, nil

	case tea.WindowSizeMsg:
		m.list.SetHeight(m.Height)
		m.list.SetWidth(m.Width)
		m.list.Styles.NoItems = noItemsStyle.Width(m.Width)

		m.conditionInput.Width = m.Width
		return m, nil

	case messages.UpdateContent:
		m.updateContent()
		return m, nil

	case tea.KeyMsg:
		var cmd tea.Cmd
		if m.addingCondition {
			var cmds []tea.Cmd
			if msg.String() == "esc" {
				m.addingCondition = false
				m.conditionInput.SetValue("")
				return m, func() tea.Msg {
					return messages.FocusedWindow(m.ID)
				}
			}
			if msg.String() == "enter" {
				m.addingCondition = false

				bp := m.list.SelectedItem().(listItem)
				_, err := m.debugger.AddConditionToBreakpoint(bp.breakpoint.ID, m.conditionInput.Value())

				if err != nil {
					m.Error = err
					return m, func() tea.Msg {
						return messages.FocusedWindow(m.ID)
					}
				}

				m.conditionInput.SetValue("")
				m.updateContent()
				return m, func() tea.Msg {
					return messages.FocusedWindow(m.ID)
				}
			}
			m.conditionInput, cmd = m.conditionInput.Update(msg)
			cmds = append(cmds, cmd)

			return m, tea.Batch(cmds...)
		}

		if !m.IsFocused {
			return m, nil
		}

		if msg.String() == "a" {
			m.CreateBreakpointNow()
			return m, nil
		}

		if msg.String() == "t" {
			m.ToggleBreakpoint()
			return m, nil
		}

		if msg.String() == "d" {
			m.ClearBreakpoint()
			return m, nil
		}

		if msg.String() == "c" {
			if m.list.SelectedItem() == nil {
				return m, nil
			}
			m.addingCondition = true
			bp := m.list.SelectedItem().(listItem)
			m.conditionInput.SetValue(bp.breakpoint.Condition)
			m.conditionInput.Focus()
			return m, func() tea.Msg {
				return messages.FocusedWindow(0)
			}
		}

		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}

	if !m.IsFocused {
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
	if m.IsFocused {
		style = listFocusedStyle
	} else {
		style = listDefaultStyle
	}

	width := m.list.Width()
	title := style.Render(fmt.Sprintf("[%d] %s", m.ID, m.title))
	titleWidth := lipgloss.Width(title)

	topBorder := style.Render("┌") + title + style.Render(strings.Repeat("─", max(width-titleWidth, 1))) + style.Render("┐")

	if m.addingCondition {
		m.conditionInput.Width = m.Width - 3
		return lipgloss.JoinVertical(lipgloss.Top,
			topBorder,
			style.
				Border(lipgloss.NormalBorder()).
				BorderTop(false).
				BorderForeground(style.GetForeground()).
				Foreground(components.ColorWhite).
				Render(m.conditionInput.View()),
		)
	}

	return lipgloss.JoinVertical(lipgloss.Top,
		topBorder,
		style.
			Border(lipgloss.NormalBorder()).
			BorderForeground(style.GetForeground()).
			BorderTop(false).
			Render(m.list.View()),
	)
}

func (m *Model) updateContent() {
	bps, err := m.debugger.GetBreakpoints()
	if err != nil {
		m.Error = fmt.Errorf("erorr updating content: %w", err)
		return
	}

	m.list.SetItems(breakpointsToListItems(bps))
}

func (m *Model) CreateBreakpointNow() {
	m.debugger.CreateBreakpointNow()
	m.updateContent()
}

func (m *Model) ToggleBreakpoint() {
	i := m.list.SelectedItem()
	if i == nil {
		return
	}
	id := i.(listItem).breakpoint.ID
	m.debugger.ToggleBreakpoint(id)
	m.updateContent()
}

func (m *Model) ClearBreakpoint() {
	i := m.list.SelectedItem()
	if i == nil {
		return
	}
	id := i.(listItem).breakpoint.ID
	m.debugger.ClearBreakpoint(id)
	m.updateContent()
	m.list.CursorUp()
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

type listDelegate struct{
	parentFocused bool
}

func (d listDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	listItem := item.(listItem)

	listItem.isFocused = m.Index() == index
	fmt.Fprint(w, listItem.Render(m.Width(), d.parentFocused))
}

func (d listDelegate) Height() int                               { return 1 }
func (d listDelegate) Spacing() int                              { return 0 }
func (d listDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

type listItem struct {
	breakpoint debugger.Breakpoint
	isFocused  bool
}

func (i listItem) FilterValue() string { return "" }

func (i listItem) Render(width int, parentFocused bool) string {
	var style lipgloss.Style
	var disabledIndicator string

	item := truncPath(i.breakpoint.Name, width-3)

	if i.breakpoint.Disabled {
		disabledIndicator = lipgloss.NewStyle().Foreground(components.ColorGrey).Render("[-] ")
	} else {
		disabledIndicator = lipgloss.NewStyle().Foreground(components.ColorGreen).Render("[+] ")
	}

	if i.breakpoint.Condition != "" {
		item += lipgloss.NewStyle().Foreground(components.ColorYellow).Render("\n\twhen: " + i.breakpoint.Condition)
	}

	if i.isFocused && parentFocused {
		style = breakpointStyleFocused
	} else {
		style = breakpointStyleDefault
	}

	breakpoint := 
	lipgloss.JoinHorizontal(lipgloss.Top, disabledIndicator, style.Render(item))

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
