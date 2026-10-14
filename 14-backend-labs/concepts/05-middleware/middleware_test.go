package middleware

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLogging(t *testing.T) {
	var logs bytes.Buffer
	logger := log.New(&logs, "", 0)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if logs.Len() != 0 {
			t.Error("completion log must come after handler")
		}
		w.WriteHeader(201)
		w.Write([]byte("saved"))
	})
	response := httptest.NewRecorder()
	request := httptest.NewRequest("POST", "/test?password=private", nil)
	Logging(logger, handler).ServeHTTP(response, request)

	if response.Code != 201 || response.Body.String() != "saved" {
		t.Error("middleware must preserve the handler response")
	}
	for _, text := range []string{"method=\"POST\"", "path=\"/test\"", "duration="} {
		if !strings.Contains(logs.String(), text) {
			t.Error("missing log detail:", text)
		}
	}
	if strings.Contains(logs.String(), "private") {
		t.Error("query must not be logged")
	}
}
