package errkit

import (
	"fmt"
	"runtime"
	"strings"
)

const (
	// callersSkip is the number of frames runtime.Callers skips internally.
	callersSkip = 2
	// stackSkipOption is the number of frames to skip when capturing stack from an Option.
	stackSkipOption = 2
)

// Frame represents a single stack frame.
type Frame struct {
	Function string `json:"function"`
	File     string `json:"file"`
	Line     int    `json:"line"`
}

// String returns a human-readable representation of the frame.
func (f Frame) String() string {
	return fmt.Sprintf("%s\n\t%s:%d", f.Function, f.File, f.Line)
}

// StackTrace represents a captured call stack.
type StackTrace []Frame

// String returns a human-readable representation of the stack trace.
func (st StackTrace) String() string {
	var b strings.Builder
	for i, f := range st {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(f.String())
	}
	return b.String()
}

// captureStack captures the current call stack, skipping the given number of frames.
func captureStack(skip int) StackTrace {
	const maxDepth = 32
	var pcs [maxDepth]uintptr
	n := runtime.Callers(skip+callersSkip, pcs[:])
	if n == 0 {
		return nil
	}

	frames := runtime.CallersFrames(pcs[:n])
	st := make(StackTrace, 0, n)
	for {
		frame, more := frames.Next()
		st = append(st, Frame{
			Function: frame.Function,
			File:     frame.File,
			Line:     frame.Line,
		})
		if !more {
			break
		}
	}
	return st
}
