package notes

import (
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestUpdateDelete(t *testing.T) {
	tests := []struct {
		name, method, id, payload string
		status                    int
	}{
		{"update", "PUT", "1", `{"text":"Updated"}`, 200},
		{"delete", "DELETE", "1", "", 204},
		{"missing update", "PUT", "99", `{"text":"Updated"}`, 404},
		{"missing delete", "DELETE", "99", "", 404},
		{"bad ID", "DELETE", "abc", "", 400},
		{"negative ID", "PUT", "-1", `{"text":"Updated"}`, 400},
		{"blank text", "PUT", "1", `{"text":" "}`, 400},
		{"database error", "DELETE", "1", "", 500},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if tt.status != 400 {
				query := "DELETE FROM backend_labs.notes"
				if tt.method == "PUT" {
					query = "UPDATE backend_labs.notes"
				}
				expect := mock.ExpectQuery(query)
				id := 1
				if tt.status == 404 {
					id = 99
				}
				if tt.method == "PUT" {
					expect.WithArgs("Updated", id)
				} else {
					expect.WithArgs(id)
				}
				if tt.status == 404 {
					expect.WillReturnError(sql.ErrNoRows)
				} else if tt.status == 500 {
					expect.WillReturnError(errors.New("database failed"))
				} else if tt.method == "PUT" {
					expect.WillReturnRows(sqlmock.NewRows([]string{"id", "text"}).AddRow(1, "Updated"))
				} else {
					expect.WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
				}
			}
			mux := http.NewServeMux()
			NewStore(db).Register(mux)
			request := httptest.NewRequest(tt.method, "/api/notes/"+tt.id, strings.NewReader(tt.payload))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if response.Code != tt.status {
				t.Error("expected", tt.status, "got", response.Code)
			}
			if tt.status == 204 && response.Body.Len() != 0 {
				t.Error("204 must have no body")
			}
			if tt.status == 200 && response.Body.String() != "{\"id\":1,\"text\":\"Updated\"}\n" {
				t.Error("unexpected note", response.Body.String())
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestNotes(t *testing.T) {
	tests := []struct {
		name, method, contentType, payload string
		status                             int
		body                               string
	}{
		{"empty list", "GET", "", "", 200, `[]`},
		{"create", "POST", "application/json", `{"text":"Learn Go"}`, 201, `{"id":1,"text":"Learn Go"}`},
		{"trim spaces", "POST", "application/json", `{"text":" Learn Go "}`, 201, `{"id":1,"text":"Learn Go"}`},
		{"missing text", "POST", "application/json", `{}`, 400, `{"error":"text is required"}`},
		{"blank text", "POST", "application/json", `{"text":" "}`, 400, `{"error":"text is required"}`},
		{"invalid JSON", "POST", "application/json", `{"text":`, 400, `{"error":"send valid JSON with string text"}`},
		{"wrong type", "POST", "application/json", `{"text":123}`, 400, `{"error":"send valid JSON with string text"}`},
		{"too large", "POST", "application/json", strings.Repeat("a", 1025), 413, `{"error":"body must not exceed 1024 bytes"}`},
		{"wrong content type", "POST", "text/plain", `{}`, 415, `{"error":"Content-Type must be application/json"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// A mock simulates SQL replies, without contacting Supabase.
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if tt.method == "GET" {
				mock.ExpectQuery("SELECT id, text FROM backend_labs.notes ORDER BY id").WillReturnRows(sqlmock.NewRows([]string{"id", "text"}))
			} else if tt.status == 201 {
				mock.ExpectQuery("INSERT INTO backend_labs.notes").WithArgs("Learn Go").WillReturnRows(sqlmock.NewRows([]string{"id", "text"}).AddRow(1, "Learn Go"))
			}
			store := NewStore(db)
			mux := http.NewServeMux()
			store.Register(mux)
			request := httptest.NewRequest(tt.method, "/api/notes", strings.NewReader(tt.payload))
			request.Header.Set("Content-Type", tt.contentType)
			response := httptest.NewRecorder()
			// Going through the router tests method/path selection too.
			mux.ServeHTTP(response, request)
			if response.Code != tt.status {
				t.Error("expected status", tt.status, "got", response.Code)
			}
			if response.Header().Get("Content-Type") != "application/json" {
				t.Error("expected JSON response")
			}
			if response.Body.String() != tt.body+"\n" {
				t.Error("expected body", tt.body, "got", response.Body.String())
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestListSavedRows(t *testing.T) {
	// Simulate rows already saved in Postgres. This is not a persistence test.
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery("SELECT id, text FROM backend_labs.notes ORDER BY id").WillReturnRows(sqlmock.NewRows([]string{"id", "text"}).AddRow(1, "First").AddRow(2, "Second"))
	store := NewStore(db)
	mux := http.NewServeMux()
	store.Register(mux)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest("GET", "/api/notes", nil))
	expected := `[{"id":1,"text":"First"},{"id":2,"text":"Second"}]` + "\n"
	if response.Body.String() != expected {
		t.Error("expected", expected, "got", response.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestUnsupportedMethod(t *testing.T) {
	mux := http.NewServeMux()
	NewStore(nil).Register(mux) // Router rejects this before accessing the DB.
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest("DELETE", "/api/notes", nil))
	// ServeMux generates this response itself, before calling our handlers.
	if response.Code != 405 || response.Header().Get("Allow") != "GET, HEAD, POST" {
		t.Error("expected 405 and Allow: GET, HEAD, POST; got", response.Code, response.Header())
	}
}

func TestDatabaseFailure(t *testing.T) {
	for _, method := range []string{"GET", "POST"} {
		t.Run(method, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			query := "SELECT id, text FROM backend_labs.notes"
			if method == "POST" {
				query = "INSERT INTO backend_labs.notes"
			}
			mock.ExpectQuery(query).WillReturnError(errors.New("database unavailable"))
			mux := http.NewServeMux()
			NewStore(db).Register(mux)
			request := httptest.NewRequest(method, "/api/notes", strings.NewReader(`{"text":"Learn Go"}`))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if response.Code != 500 {
				t.Error("expected 500, got", response.Code)
			}
			if strings.Contains(response.Body.String(), "database unavailable") {
				t.Error("internal error leaked")
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Error(err)
			}
		})
	}
}
