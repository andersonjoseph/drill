package messages

type Error error

type RefreshContent struct{}

type WindowFocused int
type ModalOpened bool

type DebuggerStepped struct{}
type DebuggerRestarted struct{}

type DebuggerBreakpointCreated struct{}
type DebuggerBreakpointToggled struct{}
type DebuggerBreakpointCleared struct{}

type BreakpointSelected int

type DebuggerStdoutReceived string
type DebuggerStderrReceived string
