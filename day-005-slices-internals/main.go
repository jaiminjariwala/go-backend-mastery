package main

import "fmt"

func main() {
	// A slice is a small HEADER (pointer + length + capacity) over a backing array.
	// Assignment copies the header, NOT the data: both share one array.
	a := []int{1, 2, 3}
	b := a     // no copy: b's header points at a's array
	b[0] = 99
	fmt.Println("a[0] =", a[0]) // 99! writing through b changed a's data

	// append may DETACH you: when full, it allocates a new bigger array.
	s := make([]int, 0, 2)
	s = append(s, 1, 2) // len 2, cap 2: completely full
	t := s              // t shares s's array (for now)
	s = append(s, 3)    // no room -> s moves to a NEW array
	t[0] = 99           // hits the OLD array, which s abandoned
	fmt.Println("s[0] =", s[0]) // 1, not 99

	// Strings are IMMUTABLE byte sequences. len = bytes, not characters.
	str := "héllo"        // é = 2 bytes in UTF-8
	fmt.Println(len(str)) // 6 bytes, not 5 characters
	fmt.Println(str[1])   // a raw byte value, not 'é'
	// str[0] = 'x'       // won't compile: strings can't be mutated

	for i, r := range str { // range decodes proper characters
		fmt.Printf("byte-index %d: character %c\n", i, r)
	}
	fmt.Println("real char count:", len([]rune(str))) // 5
}
