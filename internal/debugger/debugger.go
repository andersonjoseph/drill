package debugger

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/andersonjoseph/drill/internal/types"
	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
)

type Debugger struct {
	Client      *rpc2.RPCClient
	ready       chan string
	lcfg        api.LoadConfig
	currentFile *os.File
}

func New() (*Debugger, error) {
	d := &Debugger{
		ready: make(chan string),
		lcfg: api.LoadConfig{
			FollowPointers:     true,
			MaxVariableRecurse: 4,
			MaxStringLen:       64,
			MaxArrayValues:     16,
			MaxStructFields:    16,
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

func (d Debugger) startProcess() error {
	cmd := exec.Command("dlv", "debug", "--headless", "./cmd/test")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("error creating stdout pipe: %w", err)
	}

	cmd.Start()
	go func() {
		defer stdout.Close()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), "listening") {
				d.ready <- regexp.MustCompile(`\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?):\d{1,5}\b`).FindString(scanner.Text())
			}
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
		if currentLine == breakpointLine {
			lines.WriteString("=>")
		}
		lines.WriteString(scanner.Text() + "\n")
	}

	return lines.String(), nil
}

func (d Debugger) GetLocalVariables() ([]types.Variable, error) {
	state, err := d.Client.GetState()

	if err != nil {
		return []types.Variable{}, fmt.Errorf("error getting state: %w", err)
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
