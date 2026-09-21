// map[string]int
// Read as: "a map from strings to ints" - a lookup table. Here: word -> count.


// counts := make(map[string]int)
// Read as: "create an empty word-to-count map, with storage allocated."

// counts[word]++
// Read as: "bump the count for this word." The magic: if the word isn't in the map yet, reading it gives the zero value 0, and 0 + 1 = 1. No "key exists?" check needed.


package main

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// "countWords takes text of type string, returns a word->count map.
//  strings.Fields splits on whitespace; ToLower makes 'Go' and 'go' the same word."
func countWords(text string) map[string]int {
	counts := make(map[string]int) // empty map, ready for writes
	for _, word := range strings.Fields(strings.ToLower(text)) {
		counts[word]++ // missing key reads as 0, so this just works
	}
	return counts
}

func main() {
	counts := countWords("Go is fun and Go is fast")
	fmt.Println(counts) // map[and:1 fast:1 fun:1 go:2 is:2]

	// strings are BYTE sequences. len = bytes, not characters.
	s := "héllo"                       // é is 2 bytes in UTF-8
	fmt.Println(len(s))                // 6 (bytes)
	fmt.Println(utf8.RuneCountInString(s)) // 5 (characters)

	// range over a string decodes CHARACTERS: i = byte position, r = the rune
	for i, r := range s {
		fmt.Printf("byte-pos %d: %c\n", i, r)
	}

	delete(counts, "and")      // remove a key
	fmt.Println(counts["and"]) // 0: gone, back to the zero value
}
