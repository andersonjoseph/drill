package messages

import "github.com/charmbracelet/bubbletea"

type Error error

type RefreshContent struct{}

type WindowFocused int
type TextInputFocused bool

type DebuggerStepped struct{}
type DebuggerRestarted struct{}

type DebuggerBreakpointCreated struct {
	ID       int
	Filename string
	Line     int
}

type DebuggerBreakpointToggled struct {
	ID       int
	Filename string
	Line     int
}

type DebuggerBreakpointCleared struct {
	ID       int
	Filename string
	Line     int
}

type BreakpointSelected int

type DebuggerStdoutReceived string
type DebuggerStderrReceived string

type FileRequested struct {
	Filename string
	Line     int
}

type WindowTitleChanged struct {
	WindowID int
	Title    string
}

func DebuggerBreakpointClearedCmd(id int, file string, line int) tea.Cmd {
	return func() tea.Msg {
		return DebuggerBreakpointCleared{ID: id, Line: line, Filename: file}
	}
}

func DebuggerBreakpointToggledCmd(id int, file string, line int) tea.Cmd {
	return func() tea.Msg {
		return DebuggerBreakpointToggled{ID: id, Line: line, Filename: file}
	}
}

func DebuggerBreakpointCreatedCmd(id int, file string, line int) tea.Cmd {
	return func() tea.Msg {
		return DebuggerBreakpointCreated{ID: id, Line: line, Filename: file}
	}
}

func ErrorCmd(err error) tea.Cmd {
	return func() tea.Msg {
		if err == nil {
			return nil
		}
		return Error(err)
	}
}
