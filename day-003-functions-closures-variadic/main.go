package main

import (
	"errors"
	"fmt"
)

// multiple returns:
// divide takes 2 floats, returns a float AND an error
func divide(a, b float64) (float64, error) {
	if b == 0 {
		// 2 return values: dummy 0 + error
		return 0, errors.New("division by zero")
	}
	return a / b, nil	// nil = "no error"
}


// named returns:
// split takes an int, returns 2 named ints x and y
func split(sum int) (x, y int) {
	x = sum * 4 / 9		// assign to the named return x
	y = sum - x			// assign to the named return y
	return				// bare return = "send x and y back"
}


// variadic: any number of args, collected as a slice inside
// ...int means zero or more integers
// sum takes any number of ints, returns their total
func sum(nums ...int) int {
	total := 0
	for _, n := range nums {	// for each number n in nums
		total += n
	}
	return total
}


// higher-order function: takes a function as an argument
// filter takes nums which is a list of ints + a decider function, returns the kept ones
func filter(nums []int, keep func(int) bool) []int {
	var out []int					// start with an empty list
	for _, n := range nums {		// for each number n in nums
		if keep(n) {				// ask the decider: keep this one?
			out = append(out, n)
		}
	}
	return out
}


// closure:
// counter takes nothing, returns a function; that function counts upward
func counter() func() int {
	count := 0
	return func() int {		// return the function itself (not a call)
		count++		// bump the captured variable
		return count
	}
}


func main() {
	q, err := divide(10, 4)		// q=2.5, err=nil
	fmt.Println(q, err)
	_, err = divide(1, 0)		// _ = "don't need the quotient"
	fmt.Println("err:", err)

	x, y := split(17)
	fmt.Println(x, y)	// 7 10

	fmt.Println(sum(1, 2, 3))
	nums := []int{4, 5, 6}
	fmt.Println(sum(nums...))	// ... spreads the list into separate args -> 15


	isEven := func(n int) bool { return n%2 == 0 }	// is n even ?
	fmt.Println(filter([]int{1, 2, 3, 4, 5}, isEven))	// [2 4]

	c1 := counter()
	fmt.Println(c1(), c1(), c1())	// 1, 2, 3	(same captured count, bumped each call)
	c2 := counter()
	fmt.Println(c2())	// 1 (fresh count, independent of c1)
}