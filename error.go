package errkit

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

// Error is the core structured error type.
// It is immutable after creation — all mutation functions return a new *Error.
type Error struct {
	msg        string
	cause      error
	code       string
	fields     []Field
	stack      StackTrace
	retryable  *bool
	severity   *Severity
	httpStatus int
}

// compile-time checks
var (
	_ error          = (*Error)(nil)
	_ fmt.Formatter  = (*Error)(nil)
	_ slog.LogValuer = (*Error)(nil)
	_ json.Marshaler = (*Error)(nil)
)

// New creates a new Error with the given message and options.
func New(msg string, opts ...Option) *Error {
	e := &Error{msg: msg}
	for _, o := range opts {
		o(e)
	}
	return e
}

// Wrap wraps an existing error with a message and options.
// If err is nil, Wrap returns nil.
func Wrap(err error, msg string, opts ...Option) *Error {
	if err == nil {
		return nil
	}
	e := &Error{msg: msg, cause: err}
	for _, o := range opts {
		o(e)
	}
	return e
}

// With returns a copy of err with additional metadata fields.
// If err is not an *Error, it is wrapped first.
func With(err error, fields ...Field) *Error {
	if err == nil {
		return nil
	}
	e := clone(asError(err))
	e.fields = append(e.fields, fields...)
	return e
}

// WithCode returns a copy of err with the given error code.
func WithCode(err error, code string) *Error {
	if err == nil {
		return nil
	}
	e := clone(asError(err))
	e.code = code
	return e
}

// MarkRetryable returns a copy of err marked as retryable.
func MarkRetryable(err error) *Error {
	if err == nil {
		return nil
	}
	e := clone(asError(err))
	v := true
	e.retryable = &v
	return e
}

// MarkNotRetryable returns a copy of err explicitly marked as not retryable.
func MarkNotRetryable(err error) *Error {
	if err == nil {
		return nil
	}
	e := clone(asError(err))
	v := false
	e.retryable = &v
	return e
}

// WithSeverity returns a copy of err with the given severity.
func WithSeverity(err error, s Severity) *Error {
	if err == nil {
		return nil
	}
	e := clone(asError(err))
	e.severity = &s
	return e
}

// WithHTTP returns a copy of err with the given HTTP status code.
func WithHTTP(err error, status int) *Error {
	if err == nil {
		return nil
	}
	e := clone(asError(err))
	e.httpStatus = status
	return e
}

// WithStack returns a copy of err with a captured stack trace.
func WithStack(err error) *Error {
	if err == nil {
		return nil
	}
	e := clone(asError(err))
	e.stack = captureStack(1)
	return e
}

// ---- error interface ----

// Error returns the error message. If there is a cause, it is appended.
func (e *Error) Error() string {
	if e.cause != nil {
		return e.msg + ": " + e.cause.Error()
	}
	return e.msg
}

// Unwrap returns the underlying cause, supporting errors.Is/As.
func (e *Error) Unwrap() error {
	return e.cause
}

// ---- fmt.Formatter ----

// Format implements fmt.Formatter for verbose (%+v) and default (%v, %s) output.
func (e *Error) Format(f fmt.State, verb rune) {
	switch verb {
	case 'v':
		if f.Flag('+') {
			fmt.Fprintf(f, "%s", e.Error())
			if e.code != "" {
				fmt.Fprintf(f, "\n  code: %s", e.code)
			}
			if e.retryable != nil {
				fmt.Fprintf(f, "\n  retryable: %t", *e.retryable)
			}
			if e.severity != nil {
				fmt.Fprintf(f, "\n  severity: %s", e.severity)
			}
			if e.httpStatus != 0 {
				fmt.Fprintf(f, "\n  http: %d", e.httpStatus)
			}
			for _, field := range e.fields {
				fmt.Fprintf(f, "\n  %s: %v", field.Key, field.Value())
			}
			if len(e.stack) > 0 {
				fmt.Fprintf(f, "\n%s", e.stack.String())
			}
			return
		}
		fmt.Fprint(f, e.Error())
	case 's':
		fmt.Fprint(f, e.Error())
	case 'q':
		fmt.Fprintf(f, "%q", e.Error())
	}
}

// ---- slog.LogValuer ----

// LogValue returns a structured slog.Value for integration with log/slog.
func (e *Error) LogValue() slog.Value {
	attrs := make([]slog.Attr, 0, 4+len(e.fields))
	attrs = append(attrs, slog.String("msg", e.msg))
	if e.code != "" {
		attrs = append(attrs, slog.String("code", e.code))
	}
	if e.retryable != nil {
		attrs = append(attrs, slog.Bool("retryable", *e.retryable))
	}
	if e.severity != nil {
		attrs = append(attrs, slog.String("severity", e.severity.String()))
	}
	if e.httpStatus != 0 {
		attrs = append(attrs, slog.Int("http_status", e.httpStatus))
	}
	for _, f := range e.fields {
		attrs = append(attrs, f.SlogAttr())
	}
	if e.cause != nil {
		attrs = append(attrs, slog.String("cause", e.cause.Error()))
	}
	return slog.GroupValue(attrs...)
}

// ---- json.Marshaler ----

type jsonError struct {
	Msg        string     `json:"msg"`
	Code       string     `json:"code,omitempty"`
	Retryable  *bool      `json:"retryable,omitempty"`
	Severity   string     `json:"severity,omitempty"`
	HTTPStatus int        `json:"http_status,omitempty"`
	Fields     jsonFields `json:"fields,omitempty"`
	Cause      string     `json:"cause,omitempty"`
	Stack      StackTrace `json:"stack,omitempty"`
}

type jsonFields map[string]any

// MarshalJSON implements json.Marshaler.
func (e *Error) MarshalJSON() ([]byte, error) {
	je := jsonError{
		Msg:        e.msg,
		Code:       e.code,
		Retryable:  e.retryable,
		HTTPStatus: e.httpStatus,
		Stack:      e.stack,
	}
	if e.severity != nil {
		je.Severity = e.severity.String()
	}
	if e.cause != nil {
		je.Cause = e.cause.Error()
	}
	if len(e.fields) > 0 {
		m := make(jsonFields, len(e.fields))
		for _, f := range e.fields {
			m[f.Key] = f.Value()
		}
		je.Fields = m
	}
	return json.Marshal(je)
}

// ---- accessors ----

// Message returns the error's own message (without cause).
func (e *Error) Message() string { return e.msg }

// ErrCode returns the error code, or empty string if not set.
func (e *Error) ErrCode() string { return e.code }

// Fields returns a copy of the error's metadata fields.
func (e *Error) Fields() []Field {
	out := make([]Field, len(e.fields))
	copy(out, e.fields)
	return out
}

// Stack returns the captured stack trace, or nil.
func (e *Error) StackTrace() StackTrace { return e.stack }

// ---- query helpers (walk the chain) ----

// CodeIs checks whether any error in the chain has the given code.
func CodeIs(err error, code string) bool {
	for err != nil {
		var e *Error
		if errors.As(err, &e) && e.code == code {
			return true
		}
		err = errors.Unwrap(err)
	}
	return false
}

// GetCode returns the first error code found in the chain, or "".
func GetCode(err error) string {
	for err != nil {
		var e *Error
		if errors.As(err, &e) && e.code != "" {
			return e.code
		}
		err = errors.Unwrap(err)
	}
	return ""
}

// IsRetryable checks whether any error in the chain is marked retryable.
// Returns false if no retryable annotation is found.
func IsRetryable(err error) bool {
	for err != nil {
		var e *Error
		if errors.As(err, &e) && e.retryable != nil {
			return *e.retryable
		}
		err = errors.Unwrap(err)
	}
	return false
}

// GetSeverity returns the first severity found in the chain, or SeverityLow and false.
func GetSeverity(err error) (Severity, bool) {
	for err != nil {
		var e *Error
		if errors.As(err, &e) && e.severity != nil {
			return *e.severity, true
		}
		err = errors.Unwrap(err)
	}
	return SeverityLow, false
}

// HTTPStatus returns the HTTP status code from the first error in the chain that has one.
// Returns 500 (Internal Server Error) if none is set.
func HTTPStatus(err error) int {
	for err != nil {
		var e *Error
		if errors.As(err, &e) && e.httpStatus != 0 {
			return e.httpStatus
		}
		err = errors.Unwrap(err)
	}
	return http.StatusInternalServerError
}

// GetField returns the value of the first field with the given key found in the chain.
func GetField(err error, key string) (any, bool) {
	for err != nil {
		var e *Error
		if errors.As(err, &e) {
			for _, f := range e.fields {
				if f.Key == key {
					return f.Value(), true
				}
			}
		}
		err = errors.Unwrap(err)
	}
	return nil, false
}

// GetString returns the string value of a field by key.
func GetString(err error, key string) (string, bool) {
	v, ok := GetField(err, key)
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// GetInt returns the int value of a field by key.
func GetInt(err error, key string) (int64, bool) {
	v, ok := GetField(err, key)
	if !ok {
		return 0, false
	}
	i, ok := v.(int64)
	return i, ok
}

// ---- internal helpers ----

// asError converts any error to *Error; wraps plain errors if needed.
func asError(err error) *Error {
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return &Error{msg: err.Error(), cause: err}
}

// clone creates a shallow copy of an Error with a copied fields slice.
func clone(e *Error) *Error {
	cp := *e
	if len(e.fields) > 0 {
		cp.fields = make([]Field, len(e.fields))
		copy(cp.fields, e.fields)
	}
	return &cp
}
