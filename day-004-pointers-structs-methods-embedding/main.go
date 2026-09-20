// how to read the new syntax?

// type Rectangle struct { Width, Height float64}
// read as: Create a new type called Rectangle. It's a struct - a bundle of named fields. It holds Width and Height, both float64. A struct is Go's way of grouping related data (like a class with only fields, no inheritance).


// rect := Rectangle{Width: 3, Height: 4}
// read as: Create a rectangle, setting Width to 3 and Height to 4"


// func (r Rectangle) Area() float64
// This is a method - a function that belongs to a type. Read it as: "Define a method called Area on the Rectangle type. Inside, the rectangle itself is called r. Takes no inputs, returns a float64."
// The (r Rectangle) part is the receiver - it's just the thing the method is attached to. You call it as rect.Area() = "call Area on rect."



// full code
package main

import (
	"fmt"
	"math"
)

// create type rectangle: a bundle of Width and Height, both float64
type Rectangle struct {
	WIDTH float64
	HEIGHT float64
}

// method area on rectangle (gets a COPY). Returns WIDTH * HEIGHT
func (r Rectangle) Area() float64 {
	return r.WIDTH * r.HEIGHT	// r = "the rectangle area was called on"
}

func (r Rectangle) Perimeter() float64 {
	return 2 * (r.WIDTH + r.HEIGHT)
}

// create type Circle a bundle of Radius
type Circle struct {
	RADIUS float64
}

func (c Circle) Area() float64 {
	return math.Pi * c.RADIUS * c.RADIUS
}



// create type ColoredRectangle: it EMBEDS Rectangle (inherits its fields + methods by promotion) and adds its own Color field.
type ColoredRectangle struct {
	Rectangle
	Color string
}

// method Scale on *Rectangle (gets the ADDRESS, so it mutates the original)
func (r *Rectangle) Scale(factor float64) {
	r.WIDTH *= factor
	r.HEIGHT *= factor
}



func main() {
	rect := Rectangle{WIDTH: 3, HEIGHT: 4}
	fmt.Println("area:", rect.Area())	// call area on rectangle
	fmt.Println("perimeter:", rect.Perimeter())

	cr := ColoredRectangle{
		Rectangle: Rectangle{WIDTH: 5, HEIGHT: 6},	// embedded field init'd by type name
		Color: "red",
	}
	fmt.Println(cr.Area())	// PROMOTED: Go finds area on the embedded rectangle
	fmt.Println(cr.WIDTH)	// PROMOTED field accesss -> 5
	fmt.Println(cr.Color)	// own field -> red

	rect.Scale(2) // pointer receiver: modifies the ORIGINAL rect
	fmt.Println("scaled:", rect.WIDTH, rect.HEIGHT) // 6 8 (changed!)

	c := Circle{RADIUS: 2}
	fmt.Println("circle area:", c.Area())
}


// Structs bundle data. Methods attach behavior to a type via a receiver. Together they're Go's answer to classes - minus inheritance.

// Value receiver = copy (read-only safe). Pointer receiver = original (can mutate). When in doubt for mutating methods, pointer.

// Embedding = composition, not inheritance. You get code reuse and the cr.Area() convenience through promotion, but there's no parent/child class relationship and no polymorphism magic - just a struct inside a struct with some syntactic sugar.