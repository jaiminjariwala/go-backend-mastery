package main

import (
	"log"
	"net/http"
	"time"

	httpbasics "github.com/jaiminjariwala/go-backend-mastery/14-backend-labs/concepts/01-http-request-response"
	jsonbasics "github.com/jaiminjariwala/go-backend-mastery/14-backend-labs/concepts/02-json-validation"
)

// main takes no parameters and returns nothing. It starts the local lab server.
func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/hello", httpbasics.Hello)
	mux.HandleFunc("/api/greet", jsonbasics.Greet)

	server := &http.Server{
		Addr:              "127.0.0.1:8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Println("HTTP lesson: http://127.0.0.1:8080/api/hello?name=Jaimin")
	log.Println("JSON lesson: POST http://127.0.0.1:8080/api/greet")
	log.Fatal(server.ListenAndServe())
}
