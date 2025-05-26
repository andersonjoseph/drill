package main

import (
	"fmt"
	"runtime"
)

func main() {
	runtime.Breakpoint()
	x := 10
	fmt.Printf("x: %v\n", x)

	x = 100
	println(x)
}
