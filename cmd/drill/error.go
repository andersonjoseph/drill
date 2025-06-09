package main

import (
	"strings"

	"github.com/andersonjoseph/drill/internal/components"
	"github.com/andersonjoseph/drill/internal/messages"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var warningStyle = lipgloss.NewStyle().
	Foreground(components.ColorOrange).
	BorderForeground(components.ColorOrange)

type errMsgModel struct {
	error error
	width int
}

func New() errMsgModel {
	return errMsgModel{}
}

func (m errMsgModel) Init() tea.Cmd {
	return nil
}

func (m errMsgModel) Update(msg tea.Msg) (errMsgModel, tea.Cmd) {
	switch msg := msg.(type) {
	case messages.Error:
		m.error = msg
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	}

	return m, nil
}

func (m errMsgModel) View() string {
	if m.error == nil {
		return ""
	}

	msg := m.buildErrorMessage(m.error.Error())
	title := "Attention"
	topBorder := "┌" + title + strings.Repeat("─", max(m.width-len(title), 1)) + "┐"

	return lipgloss.JoinVertical(
		lipgloss.Top,
		warningStyle.Render(topBorder),
		warningStyle.
			Border(lipgloss.NormalBorder()).
			Width(m.width).
			BorderTop(false).
			Render(msg),
	)
}

func (m errMsgModel) buildErrorMessage(msg string) string {
	if strings.Contains(msg, "has exited with status 0") {
		return "debug session ended press r to reset or q to quit"
	}

	if strings.Contains(msg, "error evaluating expression:") {
		return "breakpoint condition failed:" + strings.Split(msg, "error evaluating expression:")[1]
	}

	return msg
}
