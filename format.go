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

func isSameStack(a, b errors.StackTrace) bool {
	return a[len(a)-2] == b[len(b)-2]
}

func trace(err error, skip int) []uintptr {
	var stacks []errors.StackTrace

	for {
		if tracer, ok := err.(tracer); ok {
			stack := tracer.StackTrace()
			if len(stacks) == 0 || !isSameStack(stack, stacks[len(stacks)-1]) {
				stacks = append(stacks, stack)
			}

			stacks[len(stacks)-1] = stack
		}

		if causer, ok := err.(causer); ok {
			err = causer.Cause()
		} else if unwrapper, ok := err.(unwrapper); ok {
			err = unwrapper.Unwrap()
		} else {
			break
		}
	}

	if len(stacks) == 0 {
		return callers(skip)
	}

	var frames int
	for _, stack := range stacks {
		frames += len(stack)
	}

	merged := make([]uintptr, 0, frames)
	for i := len(stacks) - 1; i >= 0; i-- {
		stack := stacks[i]
		for _, frame := range stack[:len(stack)-1] {
			merged = append(merged, uintptr(frame))
		}
	}

	return append(merged, uintptr(stacks[0][len(stacks[0])-1]))
}

func callers(skip int) []uintptr {
	var pcs [_MaxStackDepth]uintptr
	n := runtime.Callers(skip+3, pcs[:])
	return pcs[0:n]
}

// FormatSkip returns the stack trace embedded in the error or,
// if none was found in the error, the current stack save n frames.
func FormatSkip(err error, n int) []byte {
	var buf bytes.Buffer
	// routine id and state aren't available in pure go, so we hard-coded these
	fmt.Fprintf(&buf, "%s\ngoroutine 1 [running]:", err)

	// format each frame of the stack to match runtime.Stack's format
	for _, frame := range trace(err, n+1) {
		pc := frame - 1
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
	return string(FormatSkip(err, skip+1))
}

// FormatString is the same as Format except a string is returned instead of a byte array.
func FormatString(err error) string {
	return string(FormatSkip(err, 1))
}
