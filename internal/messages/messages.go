package messages

type Error error

type RefreshContent struct{}

type WindowFocused int
type TextInputFocused bool

type DebuggerStepped struct{}
type DebuggerRestarted struct{}

type DebuggerBreakpointCreated struct{}
type DebuggerBreakpointToggled struct{}
type DebuggerBreakpointCleared struct{}

type BreakpointSelected int

type DebuggerStdoutReceived string
type DebuggerStderrReceived string

type OpenedFile struct {
	Filename string
	Line     int
}

type WindowTitleChanged struct {
	WindowID int
	Title    string
}
