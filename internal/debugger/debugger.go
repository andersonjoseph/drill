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

	"github.com/andersonjoseph/drill/internal/types"
	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
)

type Breakpoint struct {
	ID       int
	Name     string
	Disabled bool
}

type Debugger struct {
	Client      *rpc2.RPCClient
	ready       chan string
	Output      chan string
	lcfg        api.LoadConfig
	currentFile *os.File
}

func New() (*Debugger, error) {
	d := &Debugger{
		ready:  make(chan string),
		Output: make(chan string),
		lcfg: api.LoadConfig{
			FollowPointers:     true,
			MaxVariableRecurse: 4,
			MaxStringLen:       32,
			MaxArrayValues:     8,
			MaxStructFields:    8,
		},
	}
	d.startProcess()

	select {
	case addr := <-d.ready:
		d.Client = rpc2.NewClient(addr)
	case <-time.After(time.Second * 10):
		return nil, errors.New("timeout")
	}

	return d, nil
}

func (d *Debugger) startProcess() error {
	cmd := exec.Command("dlv", "debug", "--headless", "./cmd/test")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("error creating stdout pipe: %w", err)
	}

	err = cmd.Start()
	if err != nil {
		stdout.Close()
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

			d.Output <- scanner.Text()
		}
	}()

	return nil
}

func (d *Debugger) GetCurrentFileContent(offset int) (string, error) {
	state, err := d.Client.GetState()
	if err != nil {
		return "", fmt.Errorf("error getting debugger state: %w", err)
	}

	filename := state.CurrentThread.File
	breakpointLine := state.CurrentThread.Line

	if d.currentFile == nil || d.currentFile.Name() != filename {
		if d.currentFile != nil {
			if err := d.currentFile.Close(); err != nil {
				return "", fmt.Errorf("error closing file: %s: %w", filename, err)
			}
		}

		f, err := os.Open(filename)
		if err != nil {
			return "", fmt.Errorf("error opening file: %s: %v", filename, err)
		}
		d.currentFile = f
	}

	d.currentFile.Seek(0, 0)
	scanner := bufio.NewScanner(d.currentFile)
	currentLine := 0
	startLine := max(0, breakpointLine-offset)
	endLine := breakpointLine + offset

	lines := strings.Builder{}

	for scanner.Scan() && currentLine < endLine {
		currentLine++
		if currentLine < startLine {
			continue
		}
		lines.WriteString(fmt.Sprintf("%d", currentLine))
		if currentLine == breakpointLine {
			lines.WriteString(" => ")
		}

		lines.WriteString(scanner.Text() + "\n")
	}

	return lines.String(), nil
}
func (d *Debugger) GetCurrentFilename() (string, error) {
	if d.currentFile == nil {
		return "", errors.New("error getting the current filename: currentFile is nil")
	}

	return d.currentFile.Name(), nil
}

func (d Debugger) GetLocalVariables() ([]types.Variable, error) {
	state, err := d.Client.GetState()

	if err != nil {
		return []types.Variable{}, fmt.Errorf("error getting debugger state: %w", err)
	}

	vars, err := d.Client.ListLocalVariables(
		api.EvalScope{
			GoroutineID: state.CurrentThread.GoroutineID,
		}, d.lcfg)

	if err != nil {
		return []types.Variable{}, fmt.Errorf("error getting local variables: %w", err)
	}

	localVariables := make([]types.Variable, len(vars))
	for i := range vars {
		localVariables[i] = types.Variable{
			Name:  vars[i].Name,
			Value: vars[i].SinglelineString(),
		}
	}

	return localVariables, nil
}

func (d Debugger) GetBreakpoints() ([]Breakpoint, error) {
	bps, err := d.Client.ListBreakpoints(false)
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
	bp, err := d.Client.CreateBreakpoint(&api.Breakpoint{
		Line: line,
		File: filename,
	})
	if err != nil {
		return Breakpoint{}, fmt.Errorf("error creating breakpoint: %w", err)

	}
	return apiBpToInternalBp(bp), nil
}

func (d Debugger) CreateBreakpointNow() (Breakpoint, error) {
	state, err := d.Client.GetState()
	if err != nil {
		return Breakpoint{}, fmt.Errorf("error getting debugger state: %w", err)
	}

	return d.CreateBreakpoint(state.CurrentThread.File, state.CurrentThread.Line)
}

func (d Debugger) ToggleBreakpoint(id int) error {
	_, err := d.Client.ToggleBreakpoint(id)
	if err != nil {
		return fmt.Errorf("error toggling breakpoint: %w", err)
	}

	return nil
}

func (d Debugger) ClearBreakpoint(id int) error {
	_, err := d.Client.ClearBreakpoint(id)
	if err != nil {
		return fmt.Errorf("error clearing breakpoint: %w", err)
	}

	return nil
}

func apiBpToInternalBp(bp *api.Breakpoint) Breakpoint {
	return Breakpoint{
		ID:       bp.ID,
		Name:     fmt.Sprintf("%s:%d", bp.File, bp.Line),
		Disabled: bp.Disabled,
	}
}
