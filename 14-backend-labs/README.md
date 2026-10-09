# Backend labs in Go

Small, focused exercises with explanations, runnable code, and tests. These labs use the repository's existing Go module.

## Structure

```text
14-backend-labs/
├── concepts/
│   └── 01-http-request-response/
│       ├── README.md
│       ├── handler.go
│       └── handler_test.go
├── cmd/server/main.go
└── web/README.md
```

One shared website will eventually display the concepts. Its UI is not implemented yet; starts with the HTTP endpoint and tests.

From the repository root:

```sh
go test ./14-backend-labs/...
go run ./14-backend-labs/cmd/server
```

Then, in another terminal:

```sh
curl -i 'http://127.0.0.1:8080/api/hello?name=Jaimin'
```

The learning process is explanation → implementation → tests → code review.
