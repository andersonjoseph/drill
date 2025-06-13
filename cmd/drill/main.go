package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/andersonjoseph/drill/internal/components/breakpoints"
	"github.com/andersonjoseph/drill/internal/components/callstack"
	"github.com/andersonjoseph/drill/internal/components/localvariables"
	"github.com/andersonjoseph/drill/internal/components/output"
	"github.com/andersonjoseph/drill/internal/components/sourcecode"
	"github.com/andersonjoseph/drill/internal/components/window"
	"github.com/andersonjoseph/drill/internal/debugger"
	"github.com/andersonjoseph/drill/internal/messages"
	tea "github.com/charmbracelet/bubbletea"
)

func parseEntryBreakpoint(bp string) (string, int, error) {
	breakpointAttrs := strings.Split(bp, ":")

	filename := breakpointAttrs[0]
	line, err := strconv.Atoi(breakpointAttrs[1])

	return filename, line, err
}

func main() {
	var bp string
	var autoContinue bool
	var filename string

	flag.StringVar(&filename, "f", "", "filename")
	flag.StringVar(&bp, "b", "", "create a breakpoint")
	flag.BoolVar(&autoContinue, "c", false, "auto continue to the first breakpoint")

	flag.Parse()

	debugger, err := debugger.New(filename)
	if err != nil {
		fmt.Println("Error creating debugger", err)
		os.Exit(1)
	}
	defer debugger.Close()

	localvariablesWindow := window.New(1, "Local Variables", localvariables.New(1, debugger))
	breakpointsWindow := window.New(2, "Breakpoints", breakpoints.New(2, debugger))
	callstackWindow := window.New(3, "Callstack", callstack.New(3, debugger))

	sourcecodeWindow := window.New(4, "Source Code", sourcecode.New(4, "Source Code", debugger))
	outputWindow := window.New(5, "Output", output.New(5, "Output", debugger))

	m := model{
		debugger: debugger,
		sidebar: []window.Model{
			localvariablesWindow,
			breakpointsWindow,
			callstackWindow,
		},
		sourceCode: sourcecodeWindow,
		output:     outputWindow,
	}

	if bp != "" {
		filename, line, err := parseEntryBreakpoint(bp)
		if err != nil {
			fmt.Println("Error parsing breakpoint:", err)
			os.Exit(1)
		}

		_, err = debugger.CreateBreakpoint(filename, line)
		if err != nil {
			fmt.Println("Error parsing breakpoint:", err)
			os.Exit(1)
		}
		if autoContinue {
			debugger.Continue()
			m.Update(messages.RefreshContent{})
		}
	}

	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
