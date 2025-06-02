package main

import (
	"fmt"
	"runtime"
)

func main() {
	runtime.Breakpoint()
	thelongestvariableeverlmaoooitssolarge := 10
	fmt.Printf("x: %v\n", thelongestvariableeverlmaoooitssolarge)
	x := 10
	a := 10
	b := 2349
	c := 12340
	d := 12340
	f := 10234

	thelongestvariableeverlmaoooitssolarge = 100000
	println(thelongestvariableeverlmaoooitssolarge)

	fmt.Printf("a: %v\n", a)
	fmt.Printf("b: %v\n", b)
	fmt.Printf("c: %v\n", c)
	fmt.Printf("d: %v\n", d)
	fmt.Printf("f: %v\n", f)

	fmt.Printf("x: %v\n", x)
}
