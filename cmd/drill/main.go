package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/andersonjoseph/drill/internal/components/breakpoints"
	"github.com/andersonjoseph/drill/internal/components/localvariables"
	"github.com/andersonjoseph/drill/internal/components/output"
	"github.com/andersonjoseph/drill/internal/components/sourcecode"
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
	defer debugger.Client.Disconnect(false)
	m := model{
		debugger: debugger,
		sidebar: sidebar{
			localVariables: localvariables.New(1, debugger),
			breakpoints:    breakpoints.New(2, debugger),
		},
		sourceCode: sourcecode.New(3, "Source Code", debugger),
		output:     output.New(4, "Output", debugger),
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
			<-debugger.Client.Continue()
			m.Update(messages.UpdateContent{})
		}
	}

	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
