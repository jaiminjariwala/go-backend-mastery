package main

import (
	"fmt"
	"os"
)

func main() {
	name := "world"
	if len(os.Args) > 1 {
		name = os.Args[1]
	}
	fmt.Printf("Hello, %s!\n", name)
}

// os.Args is a list of strings - everything typed on the command line after go run. 
// Position [0] is always the program's own name.
// so when you run `go run . Jaimin` then os.Args is["/tmp/go-build.../day-001", "Jaimin"]
// os.Args[0] = program name
// os.Args[1] = "Jaimin"
// that's why the code checks len(os.Args) > 1 before reading [1]
// otherwise it would crash if you pass nothing.
// go run . Jaimin -> everything after go run . gets handed to your program

// go build -o hello . -> build the program and save it as hello. it produces a binary file in the current directory.

// `gofmt -l .` -> this is Go's formmater. -l means "list files whose formatting is wrong." The . means "in the folder." If it prints nothing, your formatting is correct. If it prints a filename, run `gofmt -w .` to fix it.

// `go vet` hunts for bugs the compiler allows (lilke the %d-with-a-string trick). ./... means "this folder and every subfolder, recursively." Silence = clean.

