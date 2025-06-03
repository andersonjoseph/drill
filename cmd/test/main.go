package main

import (
	"fmt"
	"time"
)

// Custom types for testing
type Person struct {
	Name    string
	Age     int
	Email   string
	Active  bool
	Balance float64
}

type Address struct {
	Street  string
	City    string
	ZipCode string
	Country string
}

type Employee struct {
	Person
	ID         int
	Department string
	Address    Address
	Skills     []string
	Salary     *float64
}

// Interface for testing
type Speaker interface {
	Speak() string
}

func (p Person) Speak() string {
	return fmt.Sprintf("Hi, I'm %s", p.Name)
}

// Global variables for testing
var (
	globalString   = "This is a global string"
	globalInt      = 42
	globalBool     = true
	globalFloat    = 3.14159
	globalSlice    = []int{1, 2, 3, 4, 5}
	globalMap      = map[string]int{"one": 1, "two": 2, "three": 3}
	globalStruct   = Person{Name: "Global Person", Age: 30}
	globalPointer  *int
	globalNilSlice []string
	globalChan     = make(chan int, 5)
)

func main() {

	// Basic types
	// Integers of different sizes
	var int8Var int8 = 127
	var int16Var int16 = 32767
	var int32Var int32 = 2147483647
	var int64Var int64 = 9223372036854775807
	var uintVar uint = 4294967295
	var uint8Var uint8 = 255
	var uint16Var uint16 = 65535
	var uint32Var uint32 = 4294967295
	var uint64Var uint64 = 18446744073709551615

	// Floating point numbers
	var float32Var float32 = 3.4028235e+38
	var float64Var float64 = 1.7976931348623157e+308
	var negativeFloat = -123.456
	var tinyFloat = 0.000000000001

	// Complex numbers
	var complex64Var complex64 = complex(1.5, 2.5)
	var complex128Var complex128 = complex(3.14159, 2.71828)

	// Strings
	var emptyString = ""
	var shortString = "Hello"
	var longString = "This is a very long string that contains multiple words and should test how your debugger handles longer string values in the variable display"
	var multilineString = `This is
a multiline
string with
multiple lines`
	var unicodeString = "Hello 世界 🌍 🚀"
	var escapedString = "String with \"quotes\" and \ttabs\nand newlines"

	// Booleans
	var trueVar = true
	var falseVar = false

	// Runes
	var runeVar rune = 'A'
	var unicodeRune rune = '世'
	var emojiRune rune = '🚀'

	// Bytes
	var byteVar byte = 255
	var byteSlice = []byte("Hello, World!")

	// Pointers
	intValue := 42
	var intPointer = &intValue
	var nilPointer *string
	var pointerToPointer = &intPointer

	// Arrays
	var intArray [5]int = [5]int{1, 2, 3, 4, 5}
	var stringArray [3]string = [3]string{"one", "two", "three"}
	var boolArray [4]bool = [4]bool{true, false, true, false}
	var twoDArray [2][3]int = [2][3]int{{1, 2, 3}, {4, 5, 6}}

	// Slices
	var emptySlice []int
	var intSlice = []int{10, 20, 30, 40, 50}
	var stringSlice = []string{"apple", "banana", "cherry"}
	var byteSliceVar = []byte{0x48, 0x65, 0x6c, 0x6c, 0x6f}
	var sliceOfSlices = [][]int{{1, 2}, {3, 4}, {5, 6}}
	var nilSlice []string
	var largeSlice = make([]int, 1000)

	// Maps
	var emptyMap map[string]int
	var stringIntMap = map[string]int{
		"one":   1,
		"two":   2,
		"three": 3,
		"four":  4,
		"five":  5,
	}
	var intStringMap = map[int]string{
		1: "one",
		2: "two",
		3: "three",
	}
	var complexMap = map[string]Person{
		"john": {Name: "John Doe", Age: 30, Email: "john@example.com"},
		"jane": {Name: "Jane Smith", Age: 25, Email: "jane@example.com"},
	}
	var nestedMap = map[string]map[string]int{
		"scores": {"math": 90, "science": 85},
		"grades": {"A": 4, "B": 3, "C": 2},
	}

	// Structs
	var emptyPerson Person
	var person = Person{
		Name:    "Alice Johnson",
		Age:     28,
		Email:   "alice@example.com",
		Active:  true,
		Balance: 1234.56,
	}

	salary := 75000.50
	var employee = Employee{
		Person:     Person{Name: "Bob Smith", Age: 35, Email: "bob@company.com", Active: true, Balance: 5000.00},
		ID:         12345,
		Department: "Engineering",
		Address: Address{
			Street:  "123 Main St",
			City:    "San Francisco",
			ZipCode: "94105",
			Country: "USA",
		},
		Skills: []string{"Go", "Python", "JavaScript", "Docker", "Kubernetes"},
		Salary: &salary,
	}

	// Anonymous struct
	var anonStruct = struct {
		Field1 string
		Field2 int
		Field3 bool
	}{
		Field1: "Anonymous",
		Field2: 100,
		Field3: true,
	}

	// Interfaces
	var speaker Speaker = person
	var emptyInterface interface{}
	var interfaceInt interface{} = 42
	var interfaceString interface{} = "Hello, interface!"
	var interfaceSlice interface{} = []int{1, 2, 3}

	// Channels
	var nilChannel chan int
	var unbufferedChannel = make(chan string)
	var bufferedChannel = make(chan int, 10)

	// Send some values to buffered channel
	bufferedChannel <- 1
	bufferedChannel <- 2
	bufferedChannel <- 3

	// Functions
	var funcVar = func(x, y int) int { return x + y }
	var nilFunc func()

	// Time types
	var currentTime = time.Now()
	var duration = 5*time.Hour + 30*time.Minute
	var zeroTime time.Time

	// Error type
	var nilError error
	var errorVar = fmt.Errorf("this is an error message")

	// Special cases
	var (
		divByZero    = 0.0
		infinityVar  = 1.0 / divByZero
		nanVar       = 0.0 / divByZero
		veryLongName = "this_is_a_very_long_variable_name_that_might_cause_display_issues_in_the_debugger"
	)

	// Modify some values to test variable updates

	intValue = 100
	person.Age = 29
	stringSlice = append(stringSlice, "date", "elderberry")
	stringIntMap["six"] = 6
	delete(stringIntMap, "one")

	// Create circular reference
	type Node struct {
		Value int
		Next  *Node
	}
	node1 := &Node{Value: 1}
	node2 := &Node{Value: 2}
	node1.Next = node2
	node2.Next = node1 // circular reference

	// Print some values to prevent compiler optimization
	fmt.Println("=== Variable Test Output ===")
	fmt.Printf("Integers: %v, %v, %v, %v\n", int8Var, int16Var, int32Var, int64Var)
	fmt.Printf("Unsigned: %v, %v, %v, %v\n", uint8Var, uint16Var, uint32Var, uint64Var)
	fmt.Printf("Floats: %v, %v\n", float32Var, float64Var)
	fmt.Printf("Complex: %v, %v\n", complex64Var, complex128Var)
	fmt.Printf("Strings: %v, %v\n", shortString, unicodeString)
	fmt.Printf("Person: %+v\n", person)
	fmt.Printf("Employee: %+v\n", employee)
	fmt.Printf("Map: %v\n", stringIntMap)
	fmt.Printf("Slice: %v\n", intSlice)
	fmt.Printf("Interface: %v\n", speaker.Speak())
	fmt.Printf("Time: %v, Duration: %v\n", currentTime, duration)
	fmt.Printf("Error: %v\n", errorVar)
	fmt.Printf("Special: Infinity=%v, NaN=%v\n", infinityVar, nanVar)
	fmt.Printf("Channel length: %d\n", len(bufferedChannel))
	fmt.Printf("Function result: %d\n", funcVar(5, 3))

	// Keep these to prevent unused variable errors
	_ = emptyString
	_ = falseVar
	_ = runeVar
	_ = unicodeRune
	_ = emojiRune
	_ = byteVar
	_ = nilPointer
	_ = pointerToPointer
	_ = intArray
	_ = stringArray
	_ = boolArray
	_ = twoDArray
	_ = emptySlice
	_ = byteSliceVar
	_ = sliceOfSlices
	_ = nilSlice
	_ = largeSlice
	_ = emptyMap
	_ = intStringMap
	_ = complexMap
	_ = nestedMap
	_ = emptyPerson
	_ = anonStruct
	_ = emptyInterface
	_ = interfaceInt
	_ = interfaceString
	_ = interfaceSlice
	_ = nilChannel
	_ = unbufferedChannel
	_ = nilFunc
	_ = zeroTime
	_ = nilError
	_ = veryLongName
	_ = multilineString
	_ = escapedString
	_ = negativeFloat
	_ = tinyFloat
	_ = trueVar
	_ = uintVar
	_ = node1
	_ = byteSlice
	_ = longString
}
