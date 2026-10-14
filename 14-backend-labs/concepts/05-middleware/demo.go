// defines the example handlers and their routes
package middleware

import (
	_ "embed"
	"fmt"
	"net/http"
)

//go:embed index.html
var page string

func Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /concepts/middleware", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, page)
	})
	mux.HandleFunc("GET /api/middleware/success", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintln(w, "Hello! The handler did its job.")
	})
}
