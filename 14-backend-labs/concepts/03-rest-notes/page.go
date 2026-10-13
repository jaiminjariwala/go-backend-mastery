package notes

import (
	_ "embed"
	"net/http"
)

//go:embed index.html
var page string

// Page serves the local learning form; no database credentials reach the browser.
func Page(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(page))
}
