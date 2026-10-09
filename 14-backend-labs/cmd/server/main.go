package main

import (
	"log"
	"net/http"
	"time"

	httpbasics "github.com/jaiminjariwala/go-backend-mastery/14-backend-labs/concepts/01-http-request-response"
)

// main takes no parameters and returns nothing. It starts the local lab server.
func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/hello", httpbasics.Hello)

	server := &http.Server{
		Addr:              "127.0.0.1:8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Println("HTTP lesson: http://127.0.0.1:8080/api/hello?name=Jaimin")
	log.Fatal(server.ListenAndServe())
}
