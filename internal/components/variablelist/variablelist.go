package variablelist

import (
	"github.com/charmbracelet/lipgloss"
)

const (
	listWidth  int = 30
	listHeight int = 4

    colorBlack  = lipgloss.Color("0")
    colorWhite  = lipgloss.Color("15")
    colorGrey   = lipgloss.Color("7")
    colorPurple = lipgloss.Color("5")
    colorGreen  = lipgloss.Color("2")
)

type Variable struct {
	Name  string
	Value string
}

func (v Variable) renderVariableWithStyle(nameStyle, valueStyle lipgloss.Style) string {
	name := nameStyle.Render(v.Name + ": ")

	value := valueStyle.
		Width(listWidth - lipgloss.Width(name)).
		Render(v.Value)

	return name + value
}

func (v Variable) RenderVariable() string {
	nameStyle := lipgloss.NewStyle().Foreground(colorWhite)
	valueStyle := lipgloss.NewStyle().Foreground(colorGrey)
	return v.renderVariableWithStyle(nameStyle, valueStyle)
}

func (v Variable) RenderSelectedVariable() string {
	nameStyle := lipgloss.NewStyle().Foreground(colorPurple).Bold(true)
	valueStyle := lipgloss.NewStyle().Foreground(colorGreen).Bold(true)
	return v.renderVariableWithStyle(nameStyle, valueStyle)
}

func (v Variable) FilterValue() string { return "" }
