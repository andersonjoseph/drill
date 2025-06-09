package messages

type UpdateContent struct{}
type DebuggerStdout string
type DebuggerStderr string
type Restart struct{}

type FocusedWindow int
type IsFocused bool
type Error error
