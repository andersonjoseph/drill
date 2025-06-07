package main

import (
	"fmt"
	"os"
)

func main() {
	// Simple variables
	x := 10
	y := 20

	// Simple loop
	for i := 0; i < 100; i++ {
		x = x + i
		y = y - 1
		fmt.Printf("i=%d, x=%d, y=%d\n", i, x, y)
	}

	os.Stderr.WriteString("this is a stderr message")
	os.Stdout.WriteString("this is a stdout message")

	// Simple condition
	if x > 15 {
		fmt.Println("x is big")
	}

	fmt.Printf("Final: x=%d, y=%d\n", x, y)
}
