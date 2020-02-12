package fmterrors

import (
	"bytes"
	"fmt"
	"runtime"

	"github.com/pkg/errors"
)

const _MaxStackDepth = 32

type tracer interface {
	StackTrace() errors.StackTrace
}

type causer interface {
	Cause() error
}

type unwrapper interface {
	Unwrap() error
}

func trace(err error, skip int) errors.StackTrace {
	var lastTracer tracer
	for {
		if tracer, ok := err.(tracer); ok {
			lastTracer = tracer
		}

		if causer, ok := err.(causer); ok {
			err = causer.Cause()
		} else if unwrapper, ok := err.(unwrapper); ok {
			err = unwrapper.Unwrap()
		} else {
			break
		}
	}

	if lastTracer == nil {
		return callers(skip)
	}

	return lastTracer.StackTrace()
}

func callers(skip int) errors.StackTrace {
	var pcs [_MaxStackDepth]uintptr
	n := runtime.Callers(skip+3, pcs[:])

	stack := make(errors.StackTrace, n)
	for i, pc := range pcs[0:n] {
		stack[i] = errors.Frame(pc)
	}

	return stack
}

// FormatSkip returns the stack trace embedded in the error or,
// if none was found in the error, the current stack save n frames.
func FormatSkip(err error, n int) []byte {
	var buf bytes.Buffer
	// routine id and state aren't available in pure go, so we hard-coded these
	fmt.Fprintf(&buf, "%s\ngoroutine 1 [running]:", err)

	// format each frame of the stack to match runtime.Stack's format
	for _, frame := range trace(err, n) {
		pc := uintptr(frame) - 1
		fn := runtime.FuncForPC(pc)
		if fn != nil {
			file, line := fn.FileLine(pc)
			fmt.Fprintf(&buf, "\n%s()\n\t%s:%d +%#x", fn.Name(), file, line, fn.Entry())
		}
	}

	return buf.Bytes()
}

// Format returns the stack trace embedded in the error or,
// if none was found in the error, the current stack.
func Format(err error) []byte {
	return FormatSkip(err, 1)
}

// FormatSkipString is the same as FormatSkip except a string is returned instead of a byte array.
func FormatSkipString(err error, skip int) string {
	return string(FormatSkip(err, skip))
}

// FormatString is the same as Format except a string is returned instead of a byte array.
func FormatString(err error) string {
	return string(FormatSkip(err, 1))
}
