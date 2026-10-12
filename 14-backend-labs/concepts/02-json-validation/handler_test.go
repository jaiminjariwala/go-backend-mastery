package jsonbasics

import (
	"net/http/httptest" // Create in-memory requests and record responses.
	"strings"           // Turn a JSON string into a request-body reader.
	"testing"
)

func TestGreet(t *testing.T) {
	/*
		This file checks whether Greet behaves correctly.
		The loop describes how to test it.
		We write the testing steps once, then reuse them for all 14 cases.
	*/
	// Each row holds the inputs and expected results for one test.
	tests := []struct {
		name, method, contentType, payload string
		status                             int
		body                               string
		// paylod is what we send and body is what we expect back
	}{
		{"Valid", "POST", "application/json", `{"name":"Jaimin"}`, 200, `{"message":"Hello, Jaimin!"}`},
		{"TrimSpaces", "POST", "application/json", `{"name":" Jaimin "}`, 200, `{"message":"Hello, Jaimin!"}`},
		{"MissingName", "POST", "application/json", `{}`, 400, `{"error":"name is required"}`},
		{"BlankName", "POST", "application/json", `{"name":"   "}`, 400, `{"error":"name is required"}`},
		{"MalformedJSON", "POST", "application/json", `{"name":`, 400, `{"error":"send valid JSON with a string name"}`},
		{"WrongFieldType", "POST", "application/json", `{"name":123}`, 400, `{"error":"send valid JSON with a string name"}`},
		{"ExtraJSON", "POST", "application/json", `{"name":"Jay"} {"name":"Other"}`, 400, `{"error":"send valid JSON with a string name"}`},
		{"EmptyBody", "POST", "application/json", ` `, 400, `{"error":"send valid JSON with a string name"}`},
		{"Null", "POST", "application/json", `null`, 400, `{"error":"name is required"}`},
		{"TooLarge", "POST", "application/json", `{"name":"` + strings.Repeat("a", MaxBodyBytes) + `"}`, 413, `{"error":"body must not exceed 1024 bytes"}`},
		{"WrongMethod", "GET", "application/json", `{"name":"Jay"}`, 405, `{"error":"use POST"}`},
		{"WrongContentType", "POST", "text/plain", `{"name":"Jay"}`, 415, `{"error":"Content-Type must be application/json"}`},
		{"Charset", "POST", "application/json; charset=utf-8", `{"name":"Jaimin"}`, 200, `{"message":"Hello, Jaimin!"}`},
		{"Quotes", "POST", "application/json", `{"name":"\"Jay\""}`, 200, `{"message":"Hello, \"Jay\"!"}`},
	}

	// range visits each row; _ discards its index, and tt holds the row.
	for _, tt := range tests {
		// Run a subset using the row's name. Execute func(t *testing.T) as that subset.
		t.Run(tt.name, func(t *testing.T) {
			// NewReader lets the request read our JSON string as its body.
			requestBody := strings.NewReader(tt.payload)
			request := httptest.NewRequest(tt.method, "/api/greet", requestBody)
			request.Header.Set("Content-Type", tt.contentType)
			response := httptest.NewRecorder()

			// Call the handler directly, without starting a network server.
			Greet(response, request)

			if response.Code != tt.status {
				t.Error("expected status", tt.status, "got", response.Code)
			}
			headers := response.Header()
			contentType := headers.Get("Content-Type")
			if contentType != "application/json" {
				t.Error("expected JSON response, got", contentType)
			}
			// A 405 response also states which method is allowed.
			if tt.status == 405 && headers.Get("Allow") != "POST" {
				t.Error("expected Allow: POST, got", headers.Get("Allow"))
			}
			actualBody := response.Body.String()
			expectedBody := tt.body + "\n" // Fprintln adds a newline.
			if actualBody != expectedBody {
				t.Error("unexpected body:", actualBody, "expected:", expectedBody)
			}
		})
	}
}
