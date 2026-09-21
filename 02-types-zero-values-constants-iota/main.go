package main

import "fmt"

func main() {
	// zero values: Go initializes everything, no garbage
	var i int // int
	var s string	// string
	var b bool	// boolean
	var m map[string]int	// map strings to ints
	var sl []int	// slice of ints
	var p *int	// pointer to anint
	fmt.Printf("int=%d, string=%s, bool=%t, map=%v, slice=%v, ptr=%v\n", i, s, b, m, sl, p)


	// nil map: returns 0, not an error
	fmt.Println(m["missing"])	// prints 0, no panic
	m = make(map[string]int)
	m["a"] = 1
	fmt.Println(m)	// prints map[a:1]


	// shadowing trap
	x := 10
	if true {
		x := 20	// NEW variable, shadows the outer x
		fmt.Println("inner:", x)	// prints 20
	}
	fmt.Println("outer:", x)	// prints 10


	// untyped vs typed constants
	const untyped = 5
	var f float64 = untyped	// works: untyped constant adapts to context
	fmt.Println(f)
	const typed int = 5
	// var f2 float64 = typed -> it will throw a compile time type error.


	// iota: auto-incrementing counter per const block
	const (
		Sunday = iota	// 0
		Monday			// 1
		Tuesday			// 2
	)
	fmt.Println(Sunday, Monday, Tuesday)

	const (
		_ = iota				// skip 0
		KB = 1 << (10 * iota) 	// 2^10 = 1024. here iota is 1.
		MB						// 2^20 
		GB						// 2^30
		TB						// 2^40
	)
	fmt.Println(KB, MB, GB, TB)
}


// ========================================================

/*

// grouping syntax
const A = 1
const B = 2

can be written as...

const (
	A = 1
	B = 2
)

same ideas as import ( ) and var ( ) - it saves you from repeating the keyword.

*/


/*
const (
	Sunday = iota
	Monday
	Tuesday
)

is actually shorthand for:
const (
	Sunday = iota	// iota is 0 here
	Monday = iota	// iota is 1 here
	Tuesday = iota	// iota is 2 here
)

iota is simply the line number inside the const block, starting at 0.
Each new line bumps it by one.
*/


/*
<< shifts bits left. 1 << n means "take binary 1 and move it left n places," which equals 2 to the power of n
1        = 0b1           = 1
1 << 1   = 0b10          = 2
1 << 2   = 0b100         = 4
1 << 10  = 0b10000000000 = 1024


memory sizes are powers of 2, so bit shifting is the natural way to express them and since it's all constants, the compiler computes every value at compile time. Zero runtime cost. (>> is the mirror: right shift, i.e. divide by powers of 2)
*/