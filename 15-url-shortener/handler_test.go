package main

import (
	"database/sql"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestURLs(t *testing.T) {
	for _, tt := range []struct {
		url   string
		valid bool
	}{
		{"https://example.com/path?q=1", true}, {"http://example.com", true},
		{"javascript:alert(1)", false}, {"/relative", false}, {"https://user:pass@example.com", false}, {"https://", false},
	} {
		if validURL(tt.url) != tt.valid {
			t.Error("wrong validation:", tt.url)
		}
	}
}

func TestCreateAndCollisionRetry(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery("INSERT INTO url_shortener.links").WithArgs(sqlmock.AnyArg(), "https://example.com").WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("INSERT INTO url_shortener.links").WithArgs(sqlmock.AnyArg(), "https://example.com").WillReturnRows(sqlmock.NewRows([]string{"code"}).AddRow("aB3kP9xQ"))
	request := httptest.NewRequest("POST", "/api/links", strings.NewReader(`{"url":"https://example.com"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	(&App{db: db}).Routes().ServeHTTP(response, request)
	if response.Code != 201 || !strings.Contains(response.Body.String(), "aB3kP9xQ") {
		t.Error(response.Code, response.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestRedirect(t *testing.T) {
	for _, tt := range []struct {
		name   string
		err    error
		status int
	}{
		{"found", nil, 302}, {"missing", sql.ErrNoRows, 404}, {"db failure", errors.New("private details"), 500},
	} {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			expect := mock.ExpectQuery("SELECT original_url FROM url_shortener.links").WithArgs("aB3kP9xQ")
			if tt.err != nil {
				expect.WillReturnError(tt.err)
			} else {
				expect.WillReturnRows(sqlmock.NewRows([]string{"original_url"}).AddRow("https://example.com"))
			}
			response := httptest.NewRecorder()
			(&App{db: db}).Routes().ServeHTTP(response, httptest.NewRequest("GET", "/s/aB3kP9xQ", nil))
			if response.Code != tt.status {
				t.Error(response.Code)
			}
			if tt.status == 302 && response.Header().Get("Location") != "https://example.com" {
				t.Error("missing redirect destination")
			}
			if strings.Contains(response.Body.String(), "private details") {
				t.Error("error leaked")
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestInvalidInputWithoutDatabase(t *testing.T) {
	for _, body := range []string{`{"url":"javascript:alert(1)"}`, `{"url":`, `{}`} {
		request := httptest.NewRequest("POST", "/api/links", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		(&App{}).Routes().ServeHTTP(response, request)
		if response.Code != 400 {
			t.Error(response.Code)
		}
	}
}
