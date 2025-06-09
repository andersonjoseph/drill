package debugger

import (
	"bufio"
	"cmp"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/alecthomas/chroma/v2/quick"
	"github.com/andersonjoseph/drill/internal/components"
	"github.com/charmbracelet/lipgloss"
	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
)

const (
	arrowSymbol         = " 🢂 "
	breakpointDotSymbol = " ⏺ "
)

var (
	enabledBreakpointDot  = lipgloss.NewStyle().Foreground(components.ColorRed).Render(breakpointDotSymbol)
	disabledBreakpointDot = lipgloss.NewStyle().Foreground(components.ColorGrey).Render(breakpointDotSymbol)

	arrow             = lipgloss.NewStyle().Foreground(components.ColorGreen).Render(arrowSymbol)
	arrowInBreakpoint = lipgloss.NewStyle().Foreground(components.ColorRed).Render(arrowSymbol)

	lineNumberStyle = lipgloss.NewStyle().Foreground(components.ColorGrey)
)

type Variable struct {
	Name  string
	Value string
}

type Breakpoint struct {
	ID        int
	Name      string
	Line      int
	Filename  string
	Disabled  bool
	Condition string
}

type Debugger struct {
	client      *rpc2.RPCClient
	ready       chan string
	Stdout      chan string
	Stderr      chan string
	lcfg        api.LoadConfig
	currentFile *os.File
}

func New(filename string) (*Debugger, error) {
	d := &Debugger{
		ready:  make(chan string),
		Stdout: make(chan string),
		lcfg: api.LoadConfig{
			FollowPointers:     true,
			MaxVariableRecurse: 4,
			MaxStringLen:       32,
			MaxArrayValues:     8,
			MaxStructFields:    8,
		},
	}
	d.startProcess(filename)

	select {
	case addr := <-d.ready:
		d.client = rpc2.NewClient(addr)
	case <-time.After(time.Second * 10):
		return nil, errors.New("timeout")
	}

	return d, nil
}

func (d *Debugger) startProcess(filename string) error {
	cmd := exec.Command("dlv", "debug", "--headless", filename)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("error creating stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdout.Close()
		return fmt.Errorf("error creating stdout pipe: %w", err)
	}

	err = cmd.Start()
	if err != nil {
		stdout.Close()
		stderr.Close()
		return fmt.Errorf("error starting debugger process: %w", err)
	}

	addressRegex := regexp.MustCompile(`\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?):\d{1,5}\b`)

	go func() {
		defer stdout.Close()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), "listening") {
				d.ready <- addressRegex.FindString(scanner.Text())
				continue
			}

			d.Stdout <- scanner.Text()
		}
	}()

	go func() {
		defer stderr.Close()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			d.Stderr <- scanner.Text()
		}
	}()

	return nil
}

func (d *Debugger) GetCurrentFile() (*os.File, error) {
	state, err := d.client.GetState()
	if err != nil {
		return nil, fmt.Errorf("error getting current file: debugger state: %w", err)
	}

	filename := state.CurrentThread.File
	if d.currentFile == nil || d.currentFile.Name() != filename {
		if d.currentFile != nil {
			if err := d.currentFile.Close(); err != nil {
				return nil, fmt.Errorf("error getting current file content: error closing file: %s: %w", filename, err)
			}
		}

		f, err := os.Open(filename)
		if err != nil {
			return nil, fmt.Errorf("error getting current file content: error opening file: %s: %v", filename, err)
		}
		d.currentFile = f
	}

	return d.currentFile, nil
}

func (d *Debugger) GetCurrentFileContent() ([]string, error) {
	state, err := d.client.GetState()
	if err != nil {
		return nil, fmt.Errorf("error getting current file content: debugger state: %w", err)
	}

	file, err := d.GetCurrentFile()
	if err != nil {
		return nil, fmt.Errorf("error getting current file content: get current file: %w", err)
	}

	bps, err := d.GetFileBreakpoints(file.Name())
	if err != nil {
		return nil, fmt.Errorf("error getting current file content: error getting breakpoints: %v", err)
	}

	file.Seek(0, 0)
	scanner := bufio.NewScanner(file)

	line := 0
	currentLine := state.CurrentThread.Line

	var lines []string // Use slice instead of strings.Builder

	for scanner.Scan() {
		line++
		bp, isBpInLine := bps[line]

		var prefix string
		if line == currentLine {
			if isBpInLine && !bp.Disabled {
				prefix = arrowInBreakpoint
			} else {
				prefix = arrow
			}
		} else if isBpInLine {
			if bp.Disabled {
				prefix = disabledBreakpointDot
			} else {
				prefix = enabledBreakpointDot
			}
		} else {
			prefix = "   "
		}

		colorizedLine, err := colorize(scanner.Text())
		if err != nil {
			return nil, fmt.Errorf("error colorizing line: %v", err)
		}

		cleanColorizedLine := strings.ReplaceAll(colorizedLine, "\n", "")
		lines = append(lines, prefix+cleanColorizedLine)
	}

	return lines, nil
}

func (d Debugger) GetFileBreakpoints(filename string) (map[int]Breakpoint, error) {
	bpsInThisFile := make(map[int]Breakpoint, 0)
	bps, err := d.GetBreakpoints()
	if err != nil {
		return bpsInThisFile, fmt.Errorf("error getting current file content: error getting breakpoints: %s: %v", filename, err)
	}

	for _, bp := range bps {
		if bp.Filename != filename {
			continue
		}
		bpsInThisFile[bp.Line] = bp
	}

	return bpsInThisFile, nil
}

func (d *Debugger) GetCurrentFilename() (string, error) {
	f, err := d.GetCurrentFile()
	if err != nil {
		return "", fmt.Errorf("error getting the current filename: %w", err)
	}

	return f.Name(), nil
}

func (d Debugger) GetLocalVariables() ([]Variable, error) {
	state, err := d.client.GetState()
	if err != nil {
		return []Variable{}, fmt.Errorf("eerror getting local variables: debugger state: %w", err)
	}

	vars, err := d.client.ListLocalVariables(
		api.EvalScope{
			GoroutineID: state.CurrentThread.GoroutineID,
		}, d.lcfg)
	if err != nil {
		return []Variable{}, fmt.Errorf("error listing local variables: %w", err)
	}

	localVariables := make([]Variable, len(vars))
	for i := range vars {
		localVariables[i] = Variable{
			Name:  vars[i].Name,
			Value: vars[i].SinglelineString(),
		}
	}

	return localVariables, nil
}

func (d Debugger) GetBreakpoints() ([]Breakpoint, error) {
	bps, err := d.client.ListBreakpoints(false)
	if err != nil {
		return []Breakpoint{}, fmt.Errorf("error getting breakpoints: %w", err)
	}
	slices.SortFunc(bps, func(a, b *api.Breakpoint) int { return cmp.Compare(a.ID, b.ID) })

	breakpoints := make([]Breakpoint, len(bps))
	for i := range bps {
		breakpoints[i] = apiBpToInternalBp(bps[i])
	}

	return breakpoints, nil
}

func (d Debugger) CreateBreakpoint(filename string, line int) (Breakpoint, error) {
	bp, err := d.client.CreateBreakpoint(&api.Breakpoint{
		Line: line,
		File: filename,
	})
	if err != nil {
		return Breakpoint{}, fmt.Errorf("error creating breakpoint: %w", err)
	}

	return apiBpToInternalBp(bp), nil
}

func (d Debugger) CreateBreakpointNow() (Breakpoint, error) {
	state, err := d.client.GetState()
	if err != nil {
		return Breakpoint{}, fmt.Errorf("error creating breakpoint: debugger state: %w", err)
	}

	return d.CreateBreakpoint(state.CurrentThread.File, state.CurrentThread.Line)
}

func (d Debugger) AddConditionToBreakpoint(id int, cond string) (Breakpoint, error) {
	bp, err := d.client.GetBreakpoint(id)
	if err != nil {
		return Breakpoint{}, fmt.Errorf("error adding breakpoint condition: getting breakpoint: %w", err)
	}

	bp.Cond = cond

	err = d.client.AmendBreakpoint(bp)
	if err != nil {
		return Breakpoint{}, fmt.Errorf("error adding condition to breakpoint: amend breakpoint: %w", err)
	}

	return apiBpToInternalBp(bp), nil
}

func (d Debugger) ToggleBreakpoint(id int) error {
	_, err := d.client.ToggleBreakpoint(id)
	if err != nil {
		return fmt.Errorf("error toggling breakpoint: %w", err)
	}

	return nil
}

func (d Debugger) ClearBreakpoint(id int) error {
	_, err := d.client.ClearBreakpoint(id)
	if err != nil {
		return fmt.Errorf("error clearing breakpoint: %w", err)
	}

	return nil
}

func (d Debugger) Next() error {
	_, err := d.client.Next()

	if err != nil {
		return fmt.Errorf("error stepping over: %w", err)
	}

	return nil
}

func (d Debugger) Continue() {
	<-d.client.Continue()
}

func (d Debugger) Restart() error {
	_, err := d.client.Restart(false)

	if err != nil {
		return fmt.Errorf("error restarting process: %w", err)
	}

	return nil
}

func (d Debugger) Close() error {
	return fmt.Errorf("error closing debugger: %w", d.client.Disconnect(false))
}

func (d Debugger) GetCurrentLine() (int, error) {
	state, err := d.client.GetState()
	if err != nil {
		return 0, fmt.Errorf("error getting current state: %w", err)
	}

	return state.CurrentThread.Line, nil
}

func apiBpToInternalBp(bp *api.Breakpoint) Breakpoint {
	return Breakpoint{
		ID:        bp.ID,
		Name:      fmt.Sprintf("%s:%d", bp.File, bp.Line),
		Line:      bp.Line,
		Filename:  bp.File,
		Disabled:  bp.Disabled,
		Condition: bp.Cond,
	}
}

func colorize(content string) (string, error) {
	sb := strings.Builder{}

	err := quick.Highlight(&sb, content, "go", "terminal8", "native")
	if err != nil {
		return "", fmt.Errorf("error highlighting the source code: %w", err)
	}

	return sb.String(), nil
}
