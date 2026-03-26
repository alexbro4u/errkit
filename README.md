# errkit

[![Go Reference](https://pkg.go.dev/badge/github.com/alexbro4u/errkit.svg)](https://pkg.go.dev/github.com/alexbro4u/errkit)

[![Go Report Card](https://goreportcard.com/badge/github.com/alexbro4u/errkit)](https://goreportcard.com/report/github.com/alexbro4u/errkit)

Structured, extensible error handling for Go with codes, metadata, and `slog` integration.

**Subpackages** — `httpkit` and `grpckit` for HTTP/gRPC response helpers.

## Features

- **Wrap & chain** — full `errors.Is` / `errors.As` / `Unwrap` support
- **Error codes** — classify errors without parsing messages
- **Structured metadata** — typed key-value fields on every error
- **Retryable / severity flags** — drive retry logic and alerting
- **HTTP status mapping** — map errors to status codes
- **`slog.LogValuer`** — structured logging out of the box
- **`json.Marshaler`** — serialize errors for APIs, queues, logs
- **Optional stack traces** — capture only when needed
- **Immutable** — mutation functions always return a new `*Error`
- **`httpkit`** — JSON error responses, handler wrapper, panic recovery middleware
- **`grpckit`** — gRPC status conversion, interceptors, HTTP↔gRPC code mapping

## Install

```bash
go get github.com/alexbro4u/errkit
```

## Quick start

```go
package main

import (
    "fmt"
    "log/slog"
    "os"

    "github.com/alexbro4u/errkit"
)

func main() {
    // Create a structured error
    err := errkit.New("insufficient funds",
        errkit.Code("INSUFFICIENT_FUNDS"),
        errkit.HTTP(402),
        errkit.WithFields(
            errkit.String("user_id", "u_123"),
            errkit.Int("amount", 500),
        ),
    )

    // Wrap with context
    err = errkit.Wrap(err, "payment failed", errkit.Retryable())

    // Query the chain
    fmt.Println(errkit.CodeIs(err, "INSUFFICIENT_FUNDS")) // true
    fmt.Println(errkit.IsRetryable(err))                   // true
    fmt.Println(errkit.HTTPStatus(err))                    // 402

    // Structured logging
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
    logger.Error("operation failed", slog.Any("error", err))
}
```

## API overview

### Creating errors

```go
err := errkit.New("something failed")

err := errkit.New("not found",
    errkit.Code("NOT_FOUND"),
    errkit.HTTP(404),
    errkit.Retryable(),
    errkit.WithSev(errkit.SeverityHigh),
    errkit.Stack(),
    errkit.WithFields(errkit.String("id", "123")),
)
```

### Wrapping errors

```go
err := errkit.Wrap(err, "failed to fetch user")
err := errkit.Wrap(err, "service error", errkit.Code("SERVICE_ERROR"))
```

### Adding metadata after creation

All functions return a **new** error (immutability):

```go
err = errkit.With(err, errkit.String("trace_id", tid))
err = errkit.WithCode(err, "TIMEOUT")
err = errkit.MarkRetryable(err)
err = errkit.WithSeverity(err, errkit.SeverityCritical)
err = errkit.WithHTTP(err, 503)
err = errkit.WithStack(err)
```

### Querying the error chain

```go
errkit.CodeIs(err, "NOT_FOUND")       // bool — walks the chain
errkit.GetCode(err)                    // string — first code in chain
errkit.IsRetryable(err)                // bool
errkit.GetSeverity(err)                // (Severity, bool)
errkit.HTTPStatus(err)                 // int (default 500)
errkit.GetString(err, "user_id")       // (string, bool)
errkit.GetInt(err, "attempt")          // (int64, bool)
errkit.GetField(err, "key")            // (any, bool)
```

### Standard library compatibility

```go
errors.Is(err, target)   // works through the full chain
errors.As(err, &target)  // works with *errkit.Error and any wrapped type
```

### JSON serialization

```go
data, _ := json.Marshal(err)
```

```json
{
  "msg": "not found",
  "code": "NOT_FOUND",
  "http_status": 404,
  "retryable": true,
  "severity": "high",
  "fields": {"user_id": "123"},
  "cause": "db: no rows"
}
```

### slog integration

`*Error` implements `slog.LogValuer`, so structured fields appear automatically:

```go
logger.Error("op failed", slog.Any("error", err))
```

### Verbose formatting

```go
fmt.Printf("%+v\n", err)
```

```
failed to fetch user: db timeout
  code: NOT_FOUND
  retryable: true
  severity: high
  http: 404
  user_id: 123
github.com/example/main.fetchUser
    /app/main.go:42
```

## Field types

| Constructor              | Go type   |
|--------------------------|-----------|
| `errkit.String(k, v)`    | `string`  |
| `errkit.Int(k, v)`       | `int`     |
| `errkit.Int64(k, v)`     | `int64`   |
| `errkit.Bool(k, v)`      | `bool`    |
| `errkit.Float64(k, v)`   | `float64` |
| `errkit.Any(k, v)`       | `any`     |

## Severity levels

| Constant                    | String       |
|-----------------------------|--------------|
| `errkit.SeverityLow`        | `"low"`      |
| `errkit.SeverityMedium`     | `"medium"`   |
| `errkit.SeverityHigh`       | `"high"`     |
| `errkit.SeverityCritical`   | `"critical"` |

## HTTP helpers (`httpkit`)

```go
import "github.com/alexbro4u/errkit/httpkit"
```

### Write a JSON error response

```go
func getUser(w http.ResponseWriter, r *http.Request) {
    user, err := db.FindUser(r.Context(), id)
    if err != nil {
        httpkit.WriteError(w, errkit.New("user not found",
            errkit.Code("NOT_FOUND"),
            errkit.HTTP(404),
            errkit.WithFields(errkit.String("user_id", id)),
        ))
        return
    }
    json.NewEncoder(w).Encode(user)
}
```

Response:

```json
HTTP/1.1 404 Not Found
Content-Type: application/json; charset=utf-8

{"error":"user not found","code":"NOT_FOUND","fields":{"user_id":"123"}}
```

### Handler wrapper (return errors instead of handling inline)

```go
mux.Handle("/users", httpkit.Handler(func(w http.ResponseWriter, r *http.Request) error {
    user, err := svc.GetUser(r.Context(), r.URL.Query().Get("id"))
    if err != nil {
        return err // automatically writes JSON response with status code
    }
    return json.NewEncoder(w).Encode(user)
}))
```

### Panic recovery middleware

```go
mux.Handle("/", httpkit.Middleware(router))
// panics → 500 {"error":"internal server error","code":"INTERNAL"}
```

### Override status code

```go
httpkit.WriteErrorWithStatus(w, http.StatusUnprocessableEntity, err)
```

## gRPC helpers (`grpckit`)

```go
import "github.com/alexbro4u/errkit/grpckit"
```

### Convert errkit error to gRPC status error

```go
func (s *Server) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.User, error) {
    user, err := s.svc.FindUser(ctx, req.Id)
    if err != nil {
        return nil, grpckit.Error(errkit.New("user not found",
            errkit.Code("NOT_FOUND"),
            errkit.HTTP(404),
        ))
        // → gRPC status: code=NotFound, message="user not found"
    }
    return user, nil
}
```

### Server interceptors (automatic conversion)

```go
srv := grpc.NewServer(
    grpc.UnaryInterceptor(grpckit.UnaryServerInterceptor()),
    grpc.StreamInterceptor(grpckit.StreamServerInterceptor()),
)
// Any *errkit.Error returned by handlers is automatically converted
// to a gRPC status error with the correct code.
```

### Convert gRPC error back to errkit

```go
resp, err := client.GetUser(ctx, req)
if err != nil {
    ek := grpckit.FromStatus(err)
    // ek has code="NotFound", http=404
}
```

### Check gRPC error code

```go
if grpckit.IsGRPCError(err, codes.NotFound) {
    // handle not found
}
```

### HTTP ↔ gRPC code mapping

```go
grpcCode := grpckit.HTTPToGRPC(404) // codes.NotFound
httpCode := grpckit.GRPCToHTTP(codes.NotFound) // 404
```

| HTTP | gRPC |
|------|------|
| 400 | `InvalidArgument` |
| 401 | `Unauthenticated` |
| 403 | `PermissionDenied` |
| 404 | `NotFound` |
| 409 | `AlreadyExists` |
| 429 | `ResourceExhausted` |
| 500 | `Internal` |
| 501 | `Unimplemented` |
| 503 | `Unavailable` |
| 504 | `DeadlineExceeded` |

## Design decisions

- **Immutable errors** — `With*` functions never mutate, they clone. Safe for concurrent use.
- **Options pattern** — `errkit.New` and `errkit.Wrap` accept functional options. Idiomatic and extensible.
- **Optional stack traces** — stacks are expensive; capture only when you opt in via `errkit.Stack()` or `errkit.WithStack()`.
- **No reflection** — everything is statically typed.
- **No global state** — no registries, no singletons.

## License

MIT
