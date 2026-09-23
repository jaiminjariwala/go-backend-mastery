package main

import "fmt"

// "[T any]" reads as: T is a blank. Whoever CALLS the function fills it in.
// "This function takes a T and returns nothing."
func PrintMe[T any](val T) {
	fmt.Println(val)
}

func main() {
	PrintMe(42)			// T becomes int
	PrintMe("hello")	// T becomes string
	PrintMe(3.14)		// T becomes float64
	// One function, 3 types. No copy-paste.
}
