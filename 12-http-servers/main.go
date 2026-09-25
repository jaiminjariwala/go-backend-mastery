package main

import (
	"fmt"
	"net/http"
)

func main() {
	// http.HandleFunc reads as: when a request arrives for the path "/hello", run this function to handle it. The handler takes 2 things:
	// w = the reply you're building (write the response into it)
	// r = the incoming request (who asked, what they sent)
	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "hello from my server")
	})

	// "ListenAndServe" reads as: listen on port 8080, block forever serving requests.
	// Remember Part 1: the server runs EACH request in its own goroutine.
	fmt.Println("listening on :8080")
	http.ListenAndServe(":8080", nil)
}