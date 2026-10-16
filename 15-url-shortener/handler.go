package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type App struct{ db *sql.DB }
type Link struct {
	Code string `json:"code"`
	URL  string `json:"url"`
}

func (a *App) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, page)
	})
	mux.HandleFunc("POST /api/links", a.Create)
	mux.HandleFunc("GET /api/links", a.List)
	mux.HandleFunc("GET /s/{code}", a.Redirect)
	return mux
}

func reply(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(value)
}
func fail(w http.ResponseWriter, status int, message string) {
	reply(w, status, map[string]string{"error": message})
}

func validURL(raw string) bool {
	u, err := url.ParseRequestURI(raw)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Hostname() != "" && u.User == nil
}

// Six random bytes become eight URL-safe characters: 48 bits of randomness.
func newCode() (string, error) {
	var bytes [6]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes[:]), nil
}

func (a *App) Create(w http.ResponseWriter, r *http.Request) {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		fail(w, 415, "Send application/json")
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 4096))
	if err != nil {
		var oversized *http.MaxBytesError
		if errors.As(err, &oversized) {
			fail(w, 413, "Request body too large")
		} else {
			fail(w, 400, "Could not read body")
		}
		return
	}
	var input struct {
		URL string `json:"url"`
	}
	if json.Unmarshal(body, &input) != nil {
		fail(w, 400, "Send JSON with a string url")
		return
	}
	input.URL = strings.TrimSpace(input.URL)
	if len(input.URL) > 2048 || !validURL(input.URL) {
		fail(w, 400, "Use an http:// or https:// URL without embedded credentials (max 2048 bytes)")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	// The primary key arbitrates simultaneous inserts. Never overwrite on collision.
	for attempt := 0; attempt < 5; attempt++ {
		code, err := newCode()
		if err != nil {
			fail(w, 500, "Could not generate code")
			return
		}
		var saved string
		err = a.db.QueryRowContext(ctx,
			"INSERT INTO url_shortener.links (code, original_url) VALUES ($1, $2) ON CONFLICT (code) DO NOTHING RETURNING code",
			code, input.URL).Scan(&saved)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			fail(w, 500, "Could not save link; check database setup")
			return
		}
		reply(w, 201, Link{Code: saved, URL: input.URL})
		return
	}
	fail(w, 503, "Could not allocate a code; retry later")
}

func (a *App) List(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	rows, err := a.db.QueryContext(ctx, "SELECT code, original_url FROM url_shortener.links ORDER BY created_at DESC, code LIMIT 50")
	if err != nil {
		fail(w, 500, "Could not list links; check database setup")
		return
	}
	defer rows.Close()
	links := []Link{}
	for rows.Next() {
		var link Link
		if rows.Scan(&link.Code, &link.URL) != nil {
			fail(w, 500, "Could not read links")
			return
		}
		links = append(links, link)
	}
	if rows.Err() != nil {
		fail(w, 500, "Could not read links")
		return
	}
	reply(w, 200, links)
}

func (a *App) Redirect(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if len(code) != 8 || strings.IndexFunc(code, func(c rune) bool {
		return !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_')
	}) != -1 {
		http.NotFound(w, r)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	var destination string
	err := a.db.QueryRowContext(ctx, "SELECT original_url FROM url_shortener.links WHERE code = $1", code).Scan(&destination)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		fail(w, 500, "Could not look up link")
		return
	}
	if !validURL(destination) {
		fail(w, 500, "Stored URL is invalid")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, r, destination, http.StatusFound)
}
