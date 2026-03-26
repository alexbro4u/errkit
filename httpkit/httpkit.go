// Package httpkit provides HTTP helpers for errkit errors.
//
// It converts errkit errors into structured JSON responses
// with the appropriate HTTP status code.
package httpkit

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/alexbro4u/errkit"
)

// ErrorBody is the JSON body written by WriteError.
type ErrorBody struct {
	Error  string         `json:"error"`
	Code   string         `json:"code,omitempty"`
	Fields map[string]any `json:"fields,omitempty"`
}

// WriteError writes a JSON error response derived from err.
// The HTTP status is taken from errkit.HTTPStatus (defaults to 500).
// The response body is a JSON object with "error", "code", and "fields" keys.
func WriteError(w http.ResponseWriter, err error) {
	WriteErrorWithStatus(w, errkit.HTTPStatus(err), err)
}

// WriteErrorWithStatus writes a JSON error response with an explicit HTTP status,
// ignoring any status attached to the error itself.
func WriteErrorWithStatus(w http.ResponseWriter, status int, err error) {
	body := ErrorBody{
		Error: err.Error(),
		Code:  errkit.GetCode(err),
	}

	// collect fields from the outermost *errkit.Error in the chain
	var ek *errkit.Error
	if ok := errors.As(err, &ek); ok {
		fields := ek.Fields()
		if len(fields) > 0 {
			body.Fields = make(map[string]any, len(fields))
			for _, f := range fields {
				body.Fields[f.Key] = f.Value()
			}
		}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

// HandlerFunc is like http.HandlerFunc but returns an error.
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

// Handler wraps a HandlerFunc into a standard http.Handler.
// If the handler returns a non-nil error, WriteError is called automatically.
func Handler(fn HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			WriteError(w, err)
		}
	})
}

// Middleware returns an http.Handler middleware that recovers panics
// and converts them into 500 responses.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rv := recover(); rv != nil {
				err := errkit.New("internal server error",
					errkit.Code("INTERNAL"),
					errkit.HTTP(http.StatusInternalServerError),
				)
				WriteError(w, err)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
