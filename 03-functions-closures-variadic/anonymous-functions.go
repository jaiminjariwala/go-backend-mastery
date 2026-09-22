package main
import "fmt"

// conversions takes a converter function + three ints,
// returns the 3 converted ints.
func conversions(converter func(int) int, x,y,z int) (int, int, int) {
	convertedX := converter(x)
	convertedY := converter(y)
	convertedZ := converter(z)
	return convertedX, convertedY, convertedZ
}

// double takes an int, returns double it
func double(a int) int {
	return a + a
}

func demoAnonymousFunctions() {

	// using a named function: pass double by name
	newX, newY, newZ := conversions(double, 1, 2, 3)
	fmt.Println(newX, newY, newZ)

	// using an anonymous function: no name, defined inline right where it's used
	newX, newY, newZ = conversions(func(a int) int {
		return a+a
	}, 1,2,3)
	fmt.Println(newX, newY, newZ)
}