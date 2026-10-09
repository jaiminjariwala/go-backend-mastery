// handler.go does the work. handler_test.go checks whether that work is correct.
/*
	Our handler promises certain behavior.
	GET /api/hello?name=Jaimin

	should produce:
	200 OK
	Content-Type: application/json
	{ "message" : "Hello, Jaimin!" }

	But what about:
	- No name ?
	- A name containing only spaces ?
	- Quotation marks inside a name ?
	- A POST request instead of GET ?

	-> You would need to repeat those checks whenever you changed the code.
	-> Tests turn those manual checks into executable code. Run one command, and Go checks all 6 scenarios for you.
	-> Tests are useful protection, not a requirement for the handler to run.
	-> Passing tests also does not prove that every possible behavior is correct: they check the scenarios we wrote.

	basically,
	handler.go 			-> application code
	handler_test.go		-> code that checks the application code

	the suffix _test.go has a special meaning in Go.
	- normal application builds exclude these test files.
	- "go test" includes them and runs the test functions.
*/
package httpbasics

import (
	"net/http/httptest" // Create requests and record responses without a server.
	"testing"           // Report test failures.
)

// Each test checks one scenario. No table, loop, or callback.
// Go supplies t, a pointer to the testing object.
// Repetition is intentional so each test is readable on its own.

func TestHelloGreeting(t *testing.T) {
	// prepare the request. nil means no request body.
	// create an HTTP request using GET, with this target URL and no request body. Store its pointer in request.
	// the function's signature: func NewRequest(method, target string, body io.Reader) *http.Request
	request := httptest.NewRequest("GET", "/api/hello?name=Jaimin", nil)

	// Record what our handler writes.
	// Create a recorder that captures an HTTP response, and store its pointer in response
	response := httptest.NewRecorder()

	// Call Hello with our response recorder and our prepared request
	// response -> w
	// request -> r
	Hello(response, request)
  
  // if actual response status is not 200, mark this test as failed.
  // response.Code -> what the handler actually sent.
  // 200 -> what this scenario expects
	if response.Code != 200 {
	
		// t.Error marks the test as failed but continues executing the reamining statements.
		t.Error("expected status 200, got", response.Code)
	}
  
  // get the recorded response headers. read the content-type header and store its value in content type
	headers := response.Header()
	contentType := headers.Get("Content-Type")
	
	// if the handler didn't declare a JSON response, report a failure.
	if contentType != "application/json" {
		t.Error("expected application/json, got", contentType)
	}

	// Read the body as text. Fprintln adds a newline.
	actualBody := response.Body.String()
	expectedBody := `{"message":"Hello, Jaimin!"}` + "\n"
	
	// if the response text differs from the expected text, report a failure.
	if actualBody != expectedBody {
		t.Error("unexpected body:", actualBody)
	}
	// Error reports a failure but lets the test continue.
}

func TestHelloMissingName(t *testing.T) {
	// Prepare the request. nil means no request body.
	request := httptest.NewRequest("GET", "/api/hello", nil)
	// Record what our handler writes.
	response := httptest.NewRecorder()
	// Call the handler directly.
	Hello(response, request)

	if response.Code != 400 {
		t.Error("expected status 400, got", response.Code)
	}

	headers := response.Header()
	contentType := headers.Get("Content-Type")
	if contentType != "application/json" {
		t.Error("expected application/json, got", contentType)
	}

	// Read the body as text. Fprintln adds a newline.
	actualBody := response.Body.String()
	expectedBody := `{"error":"name is required"}` + "\n"
	if actualBody != expectedBody {
		t.Error("unexpected body:", actualBody)
	}
	// Error reports a failure but lets the test continue.
}

func TestHelloBlankName(t *testing.T) {
	// Prepare the request. nil means no request body.
	request := httptest.NewRequest("GET", "/api/hello?name=%20%20", nil)
	// Record what our handler writes.
	response := httptest.NewRecorder()
	// Call the handler directly.
	Hello(response, request)

	if response.Code != 400 {
		t.Error("expected status 400, got", response.Code)
	}

	headers := response.Header()
	contentType := headers.Get("Content-Type")
	if contentType != "application/json" {
		t.Error("expected application/json, got", contentType)
	}

	// Read the body as text. Fprintln adds a newline.
	actualBody := response.Body.String()
	expectedBody := `{"error":"name is required"}` + "\n"
	if actualBody != expectedBody {
		t.Error("unexpected body:", actualBody)
	}
	// Error reports a failure but lets the test continue.
}

func TestHelloTrimSpaces(t *testing.T) {
	// Prepare the request. nil means no request body.
	request := httptest.NewRequest("GET", "/api/hello?name=%20Jaimin%20", nil)
	// Record what our handler writes.
	response := httptest.NewRecorder()
	// Call the handler directly.
	Hello(response, request)

	if response.Code != 200 {
		t.Error("expected status 200, got", response.Code)
	}

	headers := response.Header()
	contentType := headers.Get("Content-Type")
	if contentType != "application/json" {
		t.Error("expected application/json, got", contentType)
	}

	// Read the body as text. Fprintln adds a newline.
	actualBody := response.Body.String()
	expectedBody := `{"message":"Hello, Jaimin!"}` + "\n"
	if actualBody != expectedBody {
		t.Error("unexpected body:", actualBody)
	}
	// Error reports a failure but lets the test continue.
}

func TestHelloQuotesInName(t *testing.T) {
	// Prepare the request. nil means no request body.
	request := httptest.NewRequest("GET", "/api/hello?name=%22Jay%22", nil)
	// Record what our handler writes.
	response := httptest.NewRecorder()
	// Call the handler directly.
	Hello(response, request)

	if response.Code != 200 {
		t.Error("expected status 200, got", response.Code)
	}

	headers := response.Header()
	contentType := headers.Get("Content-Type")
	if contentType != "application/json" {
		t.Error("expected application/json, got", contentType)
	}

	// Read the body as text. Fprintln adds a newline.
	actualBody := response.Body.String()
	expectedBody := `{"message":"Hello, \"Jay\"!"}` + "\n"
	if actualBody != expectedBody {
		t.Error("unexpected body:", actualBody)
	}
	// Error reports a failure but lets the test continue.
}

func TestHelloWrongMethod(t *testing.T) {
	// Prepare the request. nil means no request body.
	request := httptest.NewRequest("POST", "/api/hello?name=Jaimin", nil)
	// Record what our handler writes.
	response := httptest.NewRecorder()
	// Call the handler directly.
	Hello(response, request)

	if response.Code != 405 {
		t.Error("expected status 405, got", response.Code)
	}

	headers := response.Header()
	contentType := headers.Get("Content-Type")
	if contentType != "application/json" {
		t.Error("expected application/json, got", contentType)
	}
	allowedMethod := headers.Get("Allow")
	if allowedMethod != "GET" {
		t.Error("expected Allow header GET, got", allowedMethod)
	}

	// Read the body as text. Fprintln adds a newline.
	actualBody := response.Body.String()
	expectedBody := `{"error":"use GET"}` + "\n"
	if actualBody != expectedBody {
		t.Error("unexpected body:", actualBody)
	}
	// Error reports a failure but lets the test continue.
}



/*
  These tests run locally inside a Go test process on our laptop, using in-memory requests and response recorders. No HTTP server starts and no network request is send.

*/