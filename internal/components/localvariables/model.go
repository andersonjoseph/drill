package localvariables

import (
	"fmt"
	"io"

	"github.com/andersonjoseph/drill/internal/components"
	"github.com/andersonjoseph/drill/internal/debugger"
	"github.com/andersonjoseph/drill/internal/messages"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/paginator"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type variableStyle struct {
	name  lipgloss.Style
	value lipgloss.Style
}

var (
	noItemsStyle lipgloss.Style = lipgloss.NewStyle().Width(0).Foreground(components.ColorGrey)

	paginatorStyleFocused lipgloss.Style = lipgloss.NewStyle().Foreground(components.ColorGreen).PaddingRight(2)
	paginatorStyleDefault lipgloss.Style = lipgloss.NewStyle().Foreground(components.ColorWhite).PaddingRight(2)

	variableStyleDefault variableStyle = variableStyle{
		name:  lipgloss.NewStyle().Foreground(components.ColorGrey),
		value: lipgloss.NewStyle().Foreground(components.ColorGrey),
	}
	variableFocusedStyle variableStyle = variableStyle{
		name:  lipgloss.NewStyle().Foreground(components.ColorPurple).Bold(true),
		value: lipgloss.NewStyle().Foreground(components.ColorGreen).Bold(true),
	}

	listFocusedStyle lipgloss.Style = lipgloss.NewStyle().Foreground(lipgloss.Color(components.ColorGreen))
	listDefaultStyle lipgloss.Style = lipgloss.NewStyle()
	listItemStyle    lipgloss.Style = lipgloss.NewStyle()
)

type Model struct {
	ID             int
	title          string
	IsFocused      bool
	width          int
	height         int
	list           list.Model
	variableViewer VariableViewerModel
	debugger       *debugger.Debugger
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
		title:          "Local Variables",
		list:           l,
		debugger:       debugger,
		variableViewer: newVariableViewer(id),
	}
}

func (m Model) Init() tea.Cmd { return nil }
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.WindowFocused:
		m.IsFocused = int(msg) == m.ID
		m.variableViewer.setFocus(m.IsFocused)
		m.list.SetDelegate(listDelegate{parentFocused: m.IsFocused})

		if !m.IsFocused {
			m.list.Styles.PaginationStyle = paginatorStyleDefault
			return m, nil
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

		var cmd tea.Cmd
		m.variableViewer, cmd = m.variableViewer.Update(msg)
		return m, cmd

	case messages.RefreshContent, messages.DebuggerRestarted, messages.DebuggerStepped:
		if err := m.updateContent(); err != nil {
			return m, func() tea.Msg {
				return messages.Error(err)
			}
		}

		if m.variableViewer.isOpen {
			if m.list.SelectedItem() == nil {
				return m, nil
			}
			lv := m.list.SelectedItem().(listItem)
			m.variableViewer.setContent(lv.variable)
			m.variableViewer.setIsOpen(true)
		}

		return m, nil

	case tea.KeyMsg:
		if !m.IsFocused {
			return m, nil
		}

		if msg.String() == "enter" {
			if m.list.SelectedItem() == nil {
				return m, nil
			}
			lv := m.list.SelectedItem().(listItem)
			m.variableViewer.setContent(lv.variable)
			m.variableViewer.setIsOpen(true)

			return m, func() tea.Msg {
				return messages.WindowTitleChanged{WindowID: m.ID, Title: fmt.Sprintf("Inspecting %s", lv.variable.Name)}
			}
		}

		var cmd tea.Cmd
		var cmds []tea.Cmd
		if msg.String() != "esc" {
			m.list, cmd = m.list.Update(msg)
			cmds = append(cmds, cmd)
		}

		m.variableViewer, cmd = m.variableViewer.Update(msg)
		cmds = append(cmds, cmd)

		return m, tea.Batch(cmds...)
	}

	return m, nil
}

func (m Model) View() string {
	if m.variableViewer.isOpen {
		return m.variableViewer.View()
	}

	return m.list.View()
}

func (m *Model) updateContent() error {
	vars, err := m.debugger.LocalVariables()
	if err != nil {
		return fmt.Errorf("erorr updating content: %w", err)
	}

	m.list.SetItems(variablesToListItems(vars))
	return nil
}

type listDelegate struct {
	parentFocused bool
}

func (d listDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	listItem, ok := item.(listItem)
	if !ok {
		return
	}

	listItem.isFocused = m.Index() == index && d.parentFocused
	fmt.Fprint(w, listItem.Render(m.Width()))
}

func (d listDelegate) Height() int                               { return 1 }
func (d listDelegate) Spacing() int                              { return 0 }
func (d listDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

type listItem struct {
	variable  debugger.Variable
	isFocused bool
}

func (i listItem) FilterValue() string { return "" }
func (i listItem) Render(width int) string {
	var style variableStyle
	if i.isFocused {
		style = variableFocusedStyle
	} else {
		style = variableStyleDefault
	}

	name := style.name.Render(i.variable.Name)
	if i.isFocused {
		name = "▶ " + name
	}
	value := style.value.
		Render(i.variable.Value)

	return listItemStyle.
		MaxWidth(width).
		Render(name+":", value)
}

func variablesToListItems(vars []debugger.Variable) []list.Item {
	items := make([]list.Item, len(vars))

	for i := range vars {
		items[i] = listItem{
			variable: vars[i],
		}
	}

	return items
}
