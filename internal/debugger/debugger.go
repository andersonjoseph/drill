package debugger

import (
	"bufio"
	"cmp"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
)

type Variable struct {
	Name           string
	Value          string
	MultilineValue string
}

type Breakpoint struct {
	ID        int
	Name      string
	Line      int
	Filename  string
	Disabled  bool
	Condition string
}

type StackFrame struct {
	Index        int
	FunctionName string
	Filename     string
	Line         int
	Error        string
}

func newStackFrame(sf api.Stackframe, i int) StackFrame {
	return StackFrame{
		Index:        i,
		FunctionName: sf.Function.Name(),
		Filename:     sf.File,
		Line:         sf.Line,
		Error:        sf.Err,
	}
}

type outputSource int

const (
	SourceUnknown outputSource = iota
	SourceStdout
	SourceStderr
	SourceCommand
)

type Output struct {
	Content string
	Source  outputSource
}

type Debugger struct {
	client  *rpc2.RPCClient
	ready   chan string
	Output  chan Output
	lcfg    api.LoadConfig
	isReady bool
}

func New(command, filename string) (*Debugger, error) {
	d := &Debugger{
		ready:  make(chan string),
		Output: make(chan Output),
		lcfg: api.LoadConfig{
			FollowPointers:     true,
			MaxVariableRecurse: 4,
			MaxStringLen:       32,
			MaxArrayValues:     32,
			MaxStructFields:    32,
		},
	}
	if err := d.startProcess(command, filename); err != nil {
		return nil, fmt.Errorf("error starting debugger process: %w", err)
	}

	select {
	case addr := <-d.ready:
		d.client = rpc2.NewClient(addr)
	case <-time.After(time.Second * 10):
		return nil, errors.New("timeout")
	}

	return d, nil
}

func (d *Debugger) startProcess(command, filename string) error {
	cmd := exec.Command("dlv", command, filename, "--headless")

	fmt.Printf("cmd.String(): %v\n", cmd.String())

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
			if !d.isReady && strings.Contains(scanner.Text(), "listening") {
				d.ready <- addressRegex.FindString(scanner.Text())
				d.isReady = true
				continue
			}

			d.Output <- Output{
				Source:  SourceStdout,
				Content: scanner.Text(),
			}
		}
	}()

	go func() {
		defer stderr.Close()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			d.Output <- Output{
				Source:  SourceStderr,
				Content: scanner.Text(),
			}
		}
	}()

	return nil
}

func (d Debugger) FileBreakpoints(filename string) (map[int]Breakpoint, error) {
	bpsInThisFile := make(map[int]Breakpoint, 0)
	bps, err := d.Breakpoints()
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

func (d Debugger) Breakpoint(id int) (Breakpoint, error) {
	bp, err := d.client.GetBreakpoint(id)
	if err != nil {
		return Breakpoint{}, fmt.Errorf("error getting breakpoint: %w", err)
	}

	return apiBpToInternalBp(*bp), nil
}

func (d Debugger) LocalVariables() ([]Variable, error) {
	state, err := d.client.GetState()
	if err != nil {
		return []Variable{}, fmt.Errorf("eerror getting local variables: debugger state: %w", err)
	}

	scope := api.EvalScope{
		GoroutineID: state.CurrentThread.GoroutineID,
	}

	vars, err := d.client.ListLocalVariables(scope, d.lcfg)
	if err != nil {
		return []Variable{}, fmt.Errorf("error listing local variables: %w", err)
	}

	args, err := d.client.ListFunctionArgs(scope, d.lcfg)
	if err != nil {
		return []Variable{}, fmt.Errorf("error listing local args: %w", err)
	}
	vars = append(vars, args...)

	localVariables := make([]Variable, len(vars))
	for i := range vars {
		localVariables[i] = apiVarToInternalVar(vars[i])
	}

	return localVariables, nil
}

func (d Debugger) CallStack() ([]StackFrame, error) {
	state, err := d.client.GetState()
	if err != nil {
		return nil, fmt.Errorf("error getting call stack: debugger state: %w", err)
	}

	stack, err := d.client.Stacktrace(
		state.CurrentThread.GoroutineID,
		50, api.StacktraceSimple,
		&api.LoadConfig{MaxStringLen: 64, MaxStructFields: 3},
	)
	if err != nil {
		return nil, fmt.Errorf("error getting call stack: stacktrace: %w", err)
	}

	frames := make([]StackFrame, len(stack))

	for i := len(stack) - 1; i >= 0; i-- {
		frames[i] = newStackFrame(stack[i], i)
	}

	return frames, nil
}

func (d Debugger) Breakpoints() ([]Breakpoint, error) {
	bps, err := d.client.ListBreakpoints(false)
	if err != nil {
		return []Breakpoint{}, fmt.Errorf("error getting breakpoints: %w", err)
	}
	slices.SortFunc(bps, func(a, b *api.Breakpoint) int { return cmp.Compare(a.ID, b.ID) })

	breakpoints := make([]Breakpoint, len(bps))
	for i := range bps {
		breakpoints[i] = apiBpToInternalBp(*bps[i])
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

	return apiBpToInternalBp(*bp), nil
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

	return apiBpToInternalBp(*bp), nil
}

func (d Debugger) AddAliasToBreakpoint(id int, alias string) (Breakpoint, error) {
	bp, err := d.client.GetBreakpoint(id)
	if err != nil {
		return Breakpoint{}, fmt.Errorf("error adding alias to breakpoint: getting breakpoint: %w", err)
	}

	bp.Name = alias
	err = d.client.AmendBreakpoint(bp)
	if err != nil {
		return Breakpoint{}, fmt.Errorf("error adding alias to breakpoint: amend breakpoint: %w", err)
	}

	return apiBpToInternalBp(*bp), nil
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

func (d Debugger) CurrentFile() (string, int, error) {
	state, err := d.client.GetState()
	if err != nil {
		return "", 0, fmt.Errorf("error getting current state: %w", err)
	}

	return state.CurrentThread.File, state.CurrentThread.Line, nil
}

func (d Debugger) StepIn() error {
	if _, err := d.client.Step(); err != nil {
		return fmt.Errorf("error stepping in: %w", err)
	}
	return nil
}

func (d Debugger) StepOut() error {
	if _, err := d.client.StepOut(); err != nil {
		return fmt.Errorf("error stepping out: %w", err)
	}
	return nil
}

func (d Debugger) EvalVariable(expr string) (variable Variable, err error) {
	state, err := d.client.GetState()
	if err != nil {
		return variable, fmt.Errorf("error getting current state: %w", err)
	}

	scope := api.EvalScope{
		GoroutineID: state.CurrentThread.GoroutineID,
	}

	v, err := d.client.EvalVariable(scope, expr, d.lcfg)
	if err != nil {
		return variable, fmt.Errorf("error evaluating expression: %w", err)
	}

	return apiVarToInternalVar(*v), nil
}

func apiBpToInternalBp(bp api.Breakpoint) Breakpoint {
	if bp.Name == "" {
		bp.Name = fmt.Sprintf("%s:%d", bp.File, bp.Line)
	}

	return Breakpoint{
		ID:        bp.ID,
		Name:      bp.Name,
		Line:      bp.Line,
		Filename:  bp.File,
		Disabled:  bp.Disabled,
		Condition: bp.Cond,
	}
}

func apiVarToInternalVar(v api.Variable) Variable {
	return Variable{
		Name:           v.Name,
		Value:          v.SinglelineString(),
		MultilineValue: v.MultilineString(" ", "%#v"),
	}
}
