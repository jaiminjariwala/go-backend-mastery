# 01: HTTP requests and responses

## The basic idea

The browser and Go server are separate programs. Clicking a website button does not directly call a Go function. The browser sends an HTTP request; the server runs a handler and sends an HTTP response.

```text
Browser → GET /api/hello?name=Jaimin → Go handler
Browser ← 200 + JSON greeting      ← Go handler
```

The client can also be a mobile app, another backend, or curl. `localhost`/`127.0.0.1` means this computer; a port such as 8080 identifies the listening service. Our server explicitly binds to loopback, not all network interfaces.

## What a request contains

```http
GET /api/hello?name=Jaimin HTTP/1.1
Host: localhost:8080
Accept: application/json
```

- Method (`GET`): retrieve something.
- Path (`/api/hello`): select a resource/operation.
- Query (`name=Jaimin`): input in the URL.
- Headers: named metadata about the message.
- Body: optional content, for example JSON in a POST request.

## Headers, word by word

Think of a package: headers are its labels, the body is its contents.

`Accept: application/json` means "I prefer JSON back."
`Content-Type: application/json` means "the content I am sending is JSON."
These are different statements. A preference does not force the server's response format. Request headers come from the client and are not inherently trusted or secret.

```go
r.Header.Get("Accept")
```

- `r`: the incoming request pointer.
- `.`: access a field or method.
- `Header`: the request's header collection (a field).
- `Get`: call its retrieval method.
- `"Accept"`: the string argument identifying the header.

Signature: `func (h Header) Get(key string) string`.
Read: Get is a method on Header; it takes one string parameter called key and returns a string. Missing header → empty string. Header names are case-insensitive.

```go
w.Header().Set("Content-Type", "application/json")
```

Read: call Header on w with no arguments; it returns the response header collection. Call Set on that collection, passing two strings; Set returns nothing.

Equivalent:

```go
headers := w.Header()
headers.Set("Content-Type", "application/json")
```

`http.Header` is based on `map[string][]string`: string keys map to slices of strings because a header may have multiple values. Copying a map value does not copy all its entries; these variables refer to the same underlying map. Set updates the response headers.

Setting Content-Type does NOT encode JSON. We must produce a valid JSON body separately. Set headers before WriteHeader or the first body write; ordinary changes after sending headers are too late.

## Responses and status

```http
HTTP/1.1 200 OK
Content-Type: application/json

{"message":"Hello, Jaimin!"}
```

A response contains status, headers, and an optional body. Common statuses: 200 success, 201 created, 400 invalid input, 404 missing resource, 405 unsupported method, 500 unexpected server failure. A delivered 400 is still an HTTP response; a connection failure may yield no response at all.

## Handler signature and pointers

```go
func Hello(w http.ResponseWriter, r *http.Request)
```

Read: declare a function named Hello. It takes w of type http.ResponseWriter and r of type pointer to http.Request. No return type follows the parentheses, so it returns nothing. It writes the response through w instead.

Go passes arguments by value. With `*http.Request`, the copied value is a pointer, not the entire request struct. This avoids a struct copy and matches the HTTP API's required signature. A struct copy would not necessarily be independent: map/pointer fields could still reference shared data. Do not mutate an incoming request casually just because it is a pointer.

`*Person` declares a pointer type; `&person` takes an address; `*pointer` accesses the pointed-to value. Go lets `r.Method` stand for `(*r).Method` when accessing a struct field.

```go
type Person struct { Name string }
func renameCopy(p Person) { p.Name = "New" }
func renameOriginal(p *Person) { p.Name = "New" }
```

The first function receives a struct copy; changing its Name does not change the caller's Name. The second receives an address; changing that object's Name affects the caller. Both return nothing.

`http.ResponseWriter` is an interface, a contract of operations, not a response struct. Its core methods are `Header() Header`, `Write([]byte) (int, error)`, and `WriteHeader(statusCode int)`. Go supplies a value implementing them. Copying that interface does not create an independent response. A pointer to the interface is unnecessary.

## Reading handler.go

1. `headers := w.Header()`: get the outgoing headers. `headers.Set(...)` declares the JSON response format.
2. `r.Method != "GET"`: reject unsupported methods with 405, an Allow header, and a JSON error. `return` stops the handler.
3. `requestURL := r.URL`: read the request's URL field.
4. `queryParameters := requestURL.Query()`: parse that URL's query parameters.
5. `name := queryParameters.Get("name")`: read the name parameter as a string.
6. `name = strings.TrimSpace(name)`: replace name with its whitespace-trimmed version.
7. If name is empty, send 400 and return.
8. Build a greeting string, then a `map[string]string` containing the message.
9. `body, err := json.Marshal(reply)`: encode the map into JSON bytes. Check the error separately.
10. `w.WriteHeader(200)`: send the success status.
11. `fmt.Fprintln(w, string(body))`: convert the bytes to text and write it with a newline.

Backtick strings, such as the fixed JSON errors, let us include quotation marks without backslash escapes. User-provided names still go through JSON encoding.

## Reading handler_test.go

Each named test is independent. There are no anonymous functions, table-driven cases, or loops.

1. Create a request with `httptest.NewRequest`.
2. Create a response recorder with `httptest.NewRecorder`.
3. Call `Hello(response, request)`.
4. Check the recorded status and headers using ordinary if statements.
5. Read the body with `response.Body.String()` and compare it with the expected JSON text.
6. `t.Error(...)` reports a failure and continues the test.

Exact text comparison is intentionally simple for this small response. Larger JSON responses generally benefit from decoding and comparing their fields.

The earlier files, including personal comments, are preserved in `reference/handler.go.txt` and `reference/handler_test.go.txt`. These text copies are not compiled.

## Reading cmd/server/main.go

- `func main()`: the executable's entry function, no parameters and no results.
- `http.NewServeMux()`: takes no arguments, returns a pointer to a router. `mux :=` stores it.
- `mux.HandleFunc("/api/hello", httpbasics.Hello)`: takes a route pattern and handler function, returns nothing. Pass the function itself, not `Hello()`; Go calls it later with each request's arguments.
- `&http.Server{...}`: construct a server struct and take its address. Addr sets the listening address, Handler selects our router, ReadHeaderTimeout bounds header reading.
- `server.ListenAndServe()`: takes no arguments, listens and serves requests, then returns an error if it stops or cannot start. `log.Fatal(...)` logs that error and exits. Graceful shutdown is a later lesson.

## Run and test

From the repository root:

```sh
go test ./14-backend-labs/...
go run ./14-backend-labs/cmd/server
```

In another terminal:

```sh
curl -i 'http://127.0.0.1:8080/api/hello?name=Jaimin'
curl -i 'http://127.0.0.1:8080/api/hello'
curl -i -X POST 'http://127.0.0.1:8080/api/hello?name=Jaimin'
```

Expect 200, 400, and 405 respectively. curl's `-i` includes response headers. Stop the server with Ctrl+C. If 8080 is occupied, stop only your known conflicting server or change Addr; do not kill arbitrary processes.

Tests cover greeting, missing/blank names, whitespace trimming, JSON escaping, content type, and unsupported method/Allow. httptest records the handler response without opening a network port. Each separate test checks status, headers, and response text.
