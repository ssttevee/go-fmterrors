package fmterrors

import (
	"bytes"
	"fmt"
	"runtime"
	"runtime/debug"

	"github.com/pkg/errors"
)

func trace(err error) errors.StackTrace {
	type tracer interface {
		StackTrace() errors.StackTrace
	}

	type causer interface {
		Cause() error
	}

	var lastTracer tracer
	for {
		if tracer, ok := err.(tracer); ok {
			lastTracer = tracer
		}

		if causer, ok := err.(causer); ok {
			err = causer.Cause()
		} else {
			break
		}
	}

	if lastTracer == nil {
		return nil
	}

	return lastTracer.StackTrace()
}

func Format(err error) []byte {
	trace := trace(err)
	if trace == nil {
		buf := bytes.NewBufferString(err.Error() + "\n")
		buf.Write(debug.Stack())
		return buf.Bytes()
	}

	var buf bytes.Buffer
	// routine id and state aren't available in pure go, so we hard-coded these
	fmt.Fprintf(&buf, "%s\ngoroutine 1 [running]:", err)

	// format each frame of the stack to match runtime.Stack's format
	for _, frame := range trace {
		pc := uintptr(frame) - 1
		fn := runtime.FuncForPC(pc)
		if fn != nil {
			file, line := fn.FileLine(pc)
			fmt.Fprintf(&buf, "\n%s()\n\t%s:%d +%#x", fn.Name(), file, line, fn.Entry())
		}
	}

	return buf.Bytes()
}
