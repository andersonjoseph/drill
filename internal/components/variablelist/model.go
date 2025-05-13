package variablelist

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/paginator"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var paginatorStyleSelected lipgloss.Style = lipgloss.NewStyle().Foreground(colorGreen).PaddingRight(2)
var paginatorStyleDefault lipgloss.Style = lipgloss.NewStyle().Foreground(colorWhite).PaddingRight(2)

type model struct {
	list      list.Model
	title     string
	id        int
	focusedId int
}

func New(title string, id int, vars []Variable) model {
	m := model{}

	items := make([]list.Item, len(vars))

	for i := range vars {
		items[i] = vars[i]
	}

	l := list.New(items, listDelegate{
		listId:    id,
		focusedId: m.focusedId,
	}, listWidth, listHeight)
	l.SetShowHelp(false)
	l.SetShowFilter(false)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.Styles.PaginationStyle = paginatorStyleDefault
	l.Styles.NoItems = lipgloss.NewStyle().Width(listWidth)

	l.Paginator = setupPagination(len(items))

	m.id = id
	m.title = title
	m.focusedId = 1
	m.list = l

	return m
}

func setupPagination(totalItems int) paginator.Model {
    p := paginator.New()
    p.Type = paginator.Arabic
    p.PerPage = 5
    p.SetTotalPages(totalItems)
    p.ArabicFormat = lipgloss.NewStyle().
        Margin(0).Padding(0).
        Align(lipgloss.Right).
        Width(listWidth).
        Render("%d of %d ")
    return p
}

func (m model) Init() tea.Cmd { return nil }
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		i, err := strconv.Atoi(msg.String())
		if err == nil {
			m.focusedId = i
		}
	}

	if m.focusedId != m.id {
		m.list.Styles.PaginationStyle = paginatorStyleDefault
		return m, nil
	}

	m.list.Styles.PaginationStyle = paginatorStyleSelected

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)

	return m, cmd
}

func (m model) renderWithBorder(style lipgloss.Style, title string) string {
	width := m.list.Width()
	titleText := style.Render(title)
	titleWidth := lipgloss.Width(titleText)

	topBorder := style.Render("┌") + titleText + style.Render(strings.Repeat("─", width-titleWidth)) + style.Render("┐")
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

func (m model) renderList() string {
	return m.renderWithBorder(lipgloss.NewStyle(), fmt.Sprintf(" [%d] %s ", m.id, m.title))
}

func (m model) renderSelectedList() string {
	return m.renderWithBorder(lipgloss.NewStyle().Foreground(lipgloss.Color(colorGreen)),
		fmt.Sprintf(" [%d] %s ", m.id, m.title))
}

func (m model) View() string {
	if m.focusedId == m.id {
		return m.renderSelectedList()
	}

	return m.renderList()
}

type listDelegate struct {
	listId    int
	focusedId int
}

func (d listDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	variable, ok := item.(Variable)
	if !ok {
		return
	}

	var str string
	if m.Index() == index {
		str = variable.RenderSelectedVariable()
	} else {
		str = variable.RenderVariable()
	}

	fmt.Fprint(w, str)
}

func (d listDelegate) Height() int  { return 1 }
func (d listDelegate) Spacing() int { return 0 }
func (d listDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
