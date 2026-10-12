# JSON, structs, validation, and error handling

Read a name from a POST request body instead of a URL query parameter.

```http
POST /api/greet
Content-Type: application/json

{"name":"Jaimin"}
```

A successful response is 200 with `{"message":"Hello, Jaimin!"}`.

## Files

- `handler.go`: request handling and validation.
- `handler_test.go`: separate tests using in-memory requests and response recorders.
- The shared server registers this handler at `/api/greet`. Lesson 01 stays at `/api/hello`.

## Reading the code

### The request struct

```go
type GreetingRequest struct {
    Name string `json:"name"`
}
```

Read: define a type named GreetingRequest. It is a struct containing a string field named Name.

Name begins with a capital letter so the JSON package can access it. The tag tells that package to use the JSON key name. Declaring `var input GreetingRequest` creates a value whose Name field starts as the empty string.

### The handler

`func Greet(w http.ResponseWriter, r *http.Request)` takes a writer and a pointer to the request, and returns nothing. It sends its result through w, just like lesson 01.

1. Set the outgoing Content-Type to application/json.
2. Reject methods other than POST with status 405 and Allow: POST.
3. Read the incoming Content-Type through r.Header.Get.
4. ParseMediaType takes a header string and returns its main type, a parameter map, and an error. Discard the map with _. This accepts application/json with or without a charset parameter. The condition uses ||, meaning OR: either parsing failed or the type is wrong.
5. MaxBytesReader takes the response writer, request body, and maximum byte count. It returns a size-limited reader. We use a 1024-byte limit for this small lesson, not a universal production limit.
6. ReadAll takes a reader and returns its bytes and an error. This reads the entire body, but only within our small limit.
7. If reading fails, errors.As distinguishes a body-size error (413) from another read failure (400). `var sizeError *http.MaxBytesError` declares a pointer initially set to nil. `&sizeError` passes its address, allowing As to store a matching error pointer. You can treat this small block as the standard size-error check while learning the main flow.
8. Unmarshal takes JSON bytes and a destination, and returns an error. `&input` points to the struct it should fill. Unlike lesson 01's Marshal, which goes Go → JSON, Unmarshal goes JSON → Go.
9. Reject decoding failures with 400. An empty body, malformed JSON, a numeric name, or multiple JSON values fails decoding.
10. Trim input.Name and reject an empty result. Parsing checks JSON structure; validation checks whether its contents make sense for our application.
11. Build and encode the reply, then write status 200 and its JSON body.

The code deliberately uses separate variables and ordinary if statements. No helper layers, callbacks, table-driven tests, or external validation library.

## Validation choices

- Missing, null, or whitespace-only names are rejected.
- An oversized body is rejected even if it contains otherwise valid JSON.
- Unknown fields are currently ignored by Go's default Unmarshal behavior. Strict unknown-field rejection can be added later.
- A null top-level value leaves the struct empty and is rejected by name validation.
- The server manages closing incoming request bodies. The handler reads the bounded body without manually closing the server-owned request.
- ReadHeaderTimeout in the shared server limits header reading, not body reading. This lesson adds a size limit; separate body-read timeouts are a later server concern.

## Tests

Each test prepares a string reader, constructs a request, sets its incoming Content-Type, creates a response recorder, and calls Greet directly. It then checks status, outgoing headers, and body text. There is no listening server or network request during these tests.

`strings.NewReader(payload)` takes a string and returns a reader over its contents. We pass that reader instead of nil because this request has a body.

## Run

From the repository root:

```sh
go test -v ./14-backend-labs/concepts/02-json-validation
go run ./14-backend-labs/cmd/server
```

In another terminal:

```sh
curl -i http://127.0.0.1:8080/api/greet -H 'Content-Type: application/json' -d '{"name":"Jaimin"}'
curl -i http://127.0.0.1:8080/api/greet -H 'Content-Type: application/json' -d '{"name":"   "}'
curl -i http://127.0.0.1:8080/api/greet -H 'Content-Type: application/json' -d '{"name":'
```

curl's -d supplies a request body and selects POST automatically. -H adds a request header; -i shows the response headers. Stop your running server with Ctrl+C and restart it to load the new route.
