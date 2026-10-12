package jsonbasics

import (
	"encoding/json" // Convert between JSON bytes and Go values.
	"errors"        // Identify a body-size error.
	"fmt"           // Write response text.
	"io"            // Read the request body.
	"mime"          // Read Content-Type even when it includes charset=utf-8.
	"net/http"
	"strings"
)

/* 
	mime is a package included with Go. MIME stands for Multipurpose Internet Mail Extensions. Although originally associated with email, its media-type format is also used in HTTP to describe content:

	Media type				Content
	application/json		JSON
	text/plain				Plain text
	text/html				HTML
	image/png				PNG image
*/

// A struct groups related fields. Name is exported so encoding/json can set it.
// The tag maps the JSON key "name" to the Go field Name.
type GreetingRequest struct {
	Name string `json:"name"`
}

// Limit the whole request body to 1024 bytes (1 KiB).
const MaxBodyBytes = 1024

// Greet reads JSON from a POST request and writes a JSON response.
// w writes the response; r points to the incoming request.
func Greet(w http.ResponseWriter, r *http.Request) {
	headers := w.Header()
	headers.Set("Content-Type", "application/json")

	if r.Method != "POST" {
		headers.Set("Allow", "POST")
		w.WriteHeader(405)
		fmt.Fprintln(w, `{"error":"use POST"}`)
		return
	}

	/* 
		- below block checks: did the client say it is sending JSON? if not, we reject the request.
	 	- ParseMediaType returns the main type, parameters, and an error.
		- We do not need the parameters, so _ discards that result.
		- in ParseMediaType, Parse means "read structured text and separate it into meaningful pieces."
		- for this input: application/json; charset=utf-8,
		- the function returns 3 values:
			"application/json"
			map[string]string{"charset":"utf-8"}
			nil
		- parameters are additional information after the semicolon. here, charset=utf-8 describes the character encoding. so _ discards the parameters because this code does not need them.
		- err receives the parsing error, or nil if parsing succeeded.
	*/

	contentType := r.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentType)
	// if parsing failed OR the parsed media type is not JSON, run this block.
	if err != nil || mediaType != "application/json" {
		w.WriteHeader(415) // Unsupported Media Type.
		fmt.Fprintln(w, `{"error":"Content-Type must be application/json"}`)
		return
	}

	// Wrap the body so reading cannot consume unlimited data.
	// create a size-limited reader around the incoming request body and store it in limitedBody
	limitedBody := http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	body, err := io.ReadAll(limitedBody)
	if err != nil {
		// As checks whether err contains a MaxBytesError.
		// &sizeError lets As store the matching error pointer in this variable.
		var sizeError *http.MaxBytesError
		if errors.As(err, &sizeError) {
			w.WriteHeader(413) // Content Too Large.
			fmt.Fprintln(w, `{"error":"body must not exceed 1024 bytes"}`)
			return
		}
		w.WriteHeader(400)	// Bad Request
		fmt.Fprintln(w, `{"error":"could not read request body"}`)
		return
	}

	// Declare a struct with its zero value: Name starts as "".
	var input GreetingRequest
	// &input gives Unmarshal permission to update our struct.
	// Unmarshal rejects malformed JSON, wrong field types, and extra JSON values.
	err = json.Unmarshal(body, &input)
	if err != nil {
		w.WriteHeader(400)
		fmt.Fprintln(w, `{"error":"send valid JSON with a string name"}`)
		return
	}

	// Valid JSON can still contain invalid application data.
	name := strings.TrimSpace(input.Name)
	if name == "" {
		w.WriteHeader(400)
		fmt.Fprintln(w, `{"error":"name is required"}`)
		return
	}

	reply := map[string]string{
		"message": "Hello, " + name + "!",
	}
	responseBody, err := json.Marshal(reply)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, `{"error":"could not encode response"}`)
		return
	}

	w.WriteHeader(200)
	fmt.Fprintln(w, string(responseBody))
	// Fprintln's byte count/error are unused here, as in lesson 01.
}
