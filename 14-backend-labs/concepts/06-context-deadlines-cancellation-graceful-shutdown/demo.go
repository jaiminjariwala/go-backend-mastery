// This file belongs to the same package as handler.go and shutdown.go.
package contextbasics

import (
	_ "embed"  // Enable go:embed; we do not call embed functions directly.
	"fmt"      // Write text, including the HTML string, to the response writer.
	"net/http" // Provide the HTTP router, request, and response-writer types.
	"time"     // Express durations such as one second or five seconds.
)

// First, copy index.html into the program at build time, stored in page.
// The next line is a compiler directive, not an ordinary explanatory comment.
//
//go:embed index.html
var page string

// Register takes a pointer to the existing HTTP router and returns nothing.
// mux is short for multiplexer: it selects a handler using method and URL path.
// Calling Register adds routes; it does not start a server or run these tasks yet.
func Register(mux *http.ServeMux) {
	// When a GET request arrives for this page, run the function below.
	// w writes the outgoing response; r points to the incoming request.
	mux.HandleFunc("GET /concepts/context", func(w http.ResponseWriter, r *http.Request) {
		// Get the response headers, then set Content-Type to HTML with UTF-8 text.
		// This labels the body so the browser interprets it as a web page.
		// Setting the header does not generate HTML; page already contains it.
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		// Next, write page's HTML into the outgoing response, not the terminal.
		// No status was set earlier, so the first write implicitly sends 200 OK.
		fmt.Fprint(w, page)
	})
	// Task takes (work duration, allowed duration) and returns a handler function.
	// These calls create handlers now; the slow work runs when a request arrives.
	// Work takes 1 second; allow 5 seconds, so it normally finishes.
	mux.HandleFunc("GET /api/context/finish", Task(time.Second, 5*time.Second))
	// Work needs 5 seconds; allow only 2, so the deadline stops it early.
	mux.HandleFunc("GET /api/context/timeout", Task(5*time.Second, 2*time.Second))
	// Work needs 10 seconds; allow 15, leaving time to click Cancel in the browser.
	// The route name does not cancel it automatically; browser abort triggers that.
	mux.HandleFunc("GET /api/context/cancel", Task(10*time.Second, 15*time.Second))
}
