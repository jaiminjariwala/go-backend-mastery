package main
import "fmt"

func add(x, y int) int {
	return x + y
}

// mul takes 2 ints, returns their product
func mul(x, y int) int {
	return x * y
}

// aggregate takes 3 ints + an arithmetic function, applies it twice.
func aggregate(a, b, c int, arithmetic func(int, int) int) int {
	firstResult := arithmetic(a, b)
	secondResult := arithmetic(firstResult, c)
	return secondResult
}

func demoFunctionsAsValues() {
	total := aggregate(2, 3, 4, add)
	fmt.Println(total)
	product := aggregate(2, 3, 4, mul)
	fmt.Println(product)
}