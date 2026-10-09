// a handler is simply a function that handles an incoming HTTP request and writes its response.

package httpbasics

import (
	"encoding/json" // Convert Go data to JSON safely.
	"fmt"           // Write text to the response.
	"net/http"      // HTTP request and response types.
	"strings"       // Remove surrounding whitespace.
)

// Hello receives a response writer and a pointer to the request.
// It returns nothing: we send the response through w.
func Hello(w http.ResponseWriter, r *http.Request) {
	// Read the headers, then set the response format.
	headers := w.Header()

	// Set takes 2 strings -> Set(key string, value string)
	// this tells browser: interpret this response body as JSON. It does not convert anything into JSON. Encoding the body is a separate step.
	headers.Set("Content-Type", "application/json")

	
	// reject unsupported methods: if the request method is not GET, send an error response and stop.
	// r.Method: incoming method
	// GET: the accepted method
	if r.Method != "GET" {
		headers.Set("Allow", "GET")
		w.WriteHeader(405) // Method Not Allowed.

		// the backticks create a raw string literal. They let us include quotation marks without writing \"
		fmt.Fprintln(w, `{"error":"use GET"}`)

		return // Stop here instead of continuing: we have finished responding to this request. It does not shut down the server. The server can still handle other requests.
	}

	// Each variable holds one step of the URL lookup.
	requestURL := r.URL
	queryParameters := requestURL.Query()	// call the URL's Query() method. it returns query parameters as a url.Values map
	name := queryParameters.Get("name")
	name = strings.TrimSpace(name)

	if name == "" {
		w.WriteHeader(400) // Bad Request.
		fmt.Fprintln(w, `{"error":"name is required"}`)
		return
	}

	// A map stores key-value pairs, both strings in this example.
	message := "Hello, " + name + "!"
	reply := map[string]string{
		"message": message,
	}

	// Marshal returns JSON bytes and an error.
	// Encoding handles names containing quotes safely.
	body, err := json.Marshal(reply)

	// if JSON encoding failed, send an internal-server error and stop
	if err != nil {
		w.WriteHeader(500) // Internal Server Error.
		fmt.Fprintln(w, `{"error":"could not encode response"}`)
		return
	}

	// Send the status before the body. Convert the bytes to text.
	w.WriteHeader(200) // OK.
	fmt.Fprintln(w, string(body))
	// Fprintln adds a newline. Its byte count and error are unused here.
	// After writing starts, a disconnected client cannot receive a new status.
}
