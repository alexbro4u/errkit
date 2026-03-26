package errkit

// Option configures an Error during construction.
type Option func(*Error)

// Code attaches an error code.
func Code(code string) Option {
	return func(e *Error) {
		e.code = code
	}
}

// WithFields attaches metadata fields to the error.
func WithFields(fields ...Field) Option {
	return func(e *Error) {
		e.fields = append(e.fields, fields...)
	}
}

// Retryable marks the error as retryable.
func Retryable() Option {
	return func(e *Error) {
		v := true
		e.retryable = &v
	}
}

// NotRetryable explicitly marks the error as not retryable.
func NotRetryable() Option {
	return func(e *Error) {
		v := false
		e.retryable = &v
	}
}

// WithSev sets the severity of the error.
func WithSev(s Severity) Option {
	return func(e *Error) {
		e.severity = &s
	}
}

// HTTP sets the HTTP status code associated with this error.
func HTTP(status int) Option {
	return func(e *Error) {
		e.httpStatus = status
	}
}

// Stack captures the call stack at the point of error creation.
func Stack() Option {
	return func(e *Error) {
		e.stack = captureStack(2)
	}
}
