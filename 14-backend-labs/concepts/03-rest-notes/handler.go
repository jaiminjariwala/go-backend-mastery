// this file does not create a database connection. the server's main.go does that and gives this file access to it.

package notes // this code belongs to the package named notes

import (
	"context"
	"database/sql"  // run database queries and read their results
	"encoding/json" // convert between JSON and Go values
	"errors"        // recognize a body-size error
	"fmt"           // write error responses
	"io"            // read the request body
	"mime"          // parse Content-Type
	"net/http"      // handle HTTP requests and responses
	"strconv"
	"strings" // Trim whitespaces
	"time"    // express the 5 second timeout
)

// Note groups the data stored for one note. Tags control the JSON field names.
// Define a type named Note. It is a struct with an integer ID and string Text.
type Note struct {
	ID   int    `json:"id"`
	Text string `json:"text"`
}

// Store shares a database connection pool, not an in-memory list of notes.
// Define a struct named Store containing a field named db, whcih points to a sql.DB. The actual notes live in PostgreSQL on Neon.
type Store struct {
	db *sql.DB
}

// NewStore returns a pointer so requests share one store instead of copying it.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// List uses the same handler parameters as earlier lessons.
// s is a receiver: it makes List a method belonging to *Store.
// Define a method named List on a pointer to Store. Inside the method, call that Store s. Receive a response writer w and request pointer r and return nothing.
func (s *Store) List(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// Stop waiting if the client disconnects or the query takes over 5 seconds.
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	rows, err := s.db.QueryContext(ctx, "SELECT id, text FROM backend_labs.notes ORDER BY id")
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, `{"error":"could not list notes"}`)
		return
	}
	defer rows.Close() // Release query resources, including the connection.
	notes := []Note{}
	for rows.Next() {
		var note Note
		// Scan copies the current row's columns into these variables.
		if err := rows.Scan(&note.ID, &note.Text); err != nil {
			w.WriteHeader(500)
			fmt.Fprintln(w, `{"error":"could not read note"}`)
			return
		}
		notes = append(notes, note)
	}
	if err := rows.Err(); err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, `{"error":"could not list notes"}`)
		return
	}
	// An empty initialized slice encodes as [], not null.
	json.NewEncoder(w).Encode(notes)
}

// Create validates a JSON body, stores a note, and returns it with status 201.
// readText validates input shared by Create and Update.
func readText(w http.ResponseWriter, r *http.Request) (string, bool) {
	w.Header().Set("Content-Type", "application/json")
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		w.WriteHeader(415)
		fmt.Fprintln(w, `{"error":"Content-Type must be application/json"}`)
		return "", false
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1024))
	if err != nil {
		var sizeError *http.MaxBytesError
		if errors.As(err, &sizeError) {
			w.WriteHeader(413)
			fmt.Fprintln(w, `{"error":"body must not exceed 1024 bytes"}`)
			return "", false
		}
		w.WriteHeader(400)
		fmt.Fprintln(w, `{"error":"could not read request body"}`)
		return "", false
	}

	// The client supplies text; the server, not the client, assigns the ID.
	var input struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(body, &input); err != nil {
		w.WriteHeader(400)
		fmt.Fprintln(w, `{"error":"send valid JSON with string text"}`)
		return "", false
	}
	input.Text = strings.TrimSpace(input.Text)
	if input.Text == "" {
		w.WriteHeader(400)
		fmt.Fprintln(w, `{"error":"text is required"}`)
		return "", false
	}

	return input.Text, true
}

func (s *Store) Create(w http.ResponseWriter, r *http.Request) {
	text, ok := readText(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	var note Note
	// $1 is a parameter, not concatenated SQL. Postgres generates the ID.
	// RETURNING gives back the saved row; Scan fills note's fields.
	err := s.db.QueryRowContext(ctx,
		"INSERT INTO backend_labs.notes (text) VALUES ($1) RETURNING id, text",
		text,
	).Scan(&note.ID, &note.Text)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, `{"error":"could not save note"}`)
		return
	}

	w.WriteHeader(201) // Created: a new resource now exists.
	// Encoder writes JSON directly to w and adds a newline.
	// If writing fails, the response has already started; do not send a second error.
	json.NewEncoder(w).Encode(note)
}

// Register attaches methods and exact paths to the shared router.
// writeJSON shares response formatting between handlers.
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(value)
}

func readID(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		writeJSON(w, 400, map[string]string{"error": "id must be a positive integer"})
		return 0, false
	}
	return id, true
}

func (s *Store) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := readID(w, r)
	if !ok {
		return
	}
	text, ok := readText(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	var note Note
	// SQL placeholders keep values separate from SQL instructions.
	err := s.db.QueryRowContext(ctx,
		"UPDATE backend_labs.notes SET text = $1 WHERE id = $2 RETURNING id, text",
		text, id,
	).Scan(&note.ID, &note.Text)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, 404, map[string]string{"error": "note not found"})
		return
	}
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "could not update note"})
		return
	}
	writeJSON(w, 200, note)
}

func (s *Store) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := readID(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	var deletedID int
	// No returned row means that ID did not exist.
	err := s.db.QueryRowContext(ctx,
		"DELETE FROM backend_labs.notes WHERE id = $1 RETURNING id", id,
	).Scan(&deletedID)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, 404, map[string]string{"error": "note not found"})
		return
	}
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "could not delete note"})
		return
	}
	w.WriteHeader(204) // Successful deletion; no response body.
}

func (s *Store) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/notes", s.List)
	mux.HandleFunc("POST /api/notes", s.Create)
	mux.HandleFunc("PUT /api/notes/{id}", s.Update)
	mux.HandleFunc("DELETE /api/notes/{id}", s.Delete)
}
