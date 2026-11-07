package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The simulator API must return the state after each accepted action, allowing
// the UI to display one consistent observation rather than a partial mutation.
func TestAPISnapshotAndActions(t *testing.T) {
	h := newHandler()
	for i := 0; i < 20; i++ {
		r := httptest.NewRequest("POST", "/api/cluster", strings.NewReader(`{"Action":"step"}`))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	r := httptest.NewRequest("POST", "/api/cluster", strings.NewReader(`{"Action":"write","Key":"x","Value":"40"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	var result struct {
		Tick     int
		Progress *WriteProgress
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Tick != 20 || result.Progress == nil || result.Progress.Committed {
		t.Fatal("unexpected proposal snapshot", w.Body.String())
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("snapshots must not be cached")
	}
}

func TestAPIRejectsInvalidRequests(t *testing.T) {
	h := newHandler()
	for _, tt := range []struct {
		body, content string
		code          int
	}{
		{`{"Action":"step"}`, "text/plain", 415},
		{`{`, "application/json", 400},
		{`{"Action":"toggle","ID":99}`, "application/json", 400},
		{`{"Action":"unknown"}`, "application/json", 400},
	} {
		r := httptest.NewRequest("POST", "/api/cluster", strings.NewReader(tt.body))
		r.Header.Set("Content-Type", tt.content)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tt.code {
			t.Fatalf("got %d for %s", w.Code, tt.body)
		}
	}
}
