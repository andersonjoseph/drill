# Drill

![Drill](https://raw.githubusercontent.com/andersonjoseph/drill/refs/heads/main/drill-gopher.png)

Drill is a terminal-based debugger designed to provide a lightweight and simple debugging experience. While it is not yet stable and lacks many features found in full-featured debuggers, Drill has been a great alternative to print debugging and fits seamlessly into my workflows (terminal-based tools)

---

## Features

- **Terminal User Interface (TUI)**: Built using [Charmbracelet's Bubbletea](https://github.com/charmbracelet/bubbletea) and friends
- **Breakpoint Management**: Set, toggle, and delete breakpoints with ease.
- **Callstack Navigation**: View and navigate through the callstack during execution.
- **Variable Inspection**: Inspect local variables at runtime.

---

## Current Limitations

Drill is still in its early stages of development and is **not stable**. It lacks many features that other full-featured debuggers provide, such as:

- Comprehensive test case support (work in progress).
- Some debugging capabilities like watchpoints, or remote debugging.
- Robust error handling (there are still some bugs here and there to fix).

(However. Drill has become my go-to debugging tool for now. It effectively meets my current needs.)

---

## Why I Built Drill

Drill was created as a learning project to explore building TUIs with [Charmbracelet's Bubbletea](https://github.com/charmbracelet/bubbletea) and related tools. Inspired by [gdlv](https://github.com/aarzilli/gdlv) and powered by [Delve](https://github.com/go-delve/delve)

---

## Future Plans

- **Test Case Support**: I plan to add support for most of the test cases described in [aarzilli/delve_client_testing](https://github.com/aarzilli/delve_client_testing).
- **Bug Fixes**: Fix some of the annoying bugs.
- **Feature Enhancements**: Gradually adding more advanced debugging features to make Drill more robust.

---

## Acknowledgments

A huge thank you to the following projects and their maintainers for their inspiration and contributions:

- [Charmbracelet's Bubbletea](https://github.com/charmbracelet/bubbletea) & Friends: An amazing framework for building TUIs.
- [gdlv](https://github.com/aarzilli/gdlv): The main inspiration for Drill.
- [Delve](https://github.com/go-delve/delve): The backend whose source code was a pleasure to read and learn from.
