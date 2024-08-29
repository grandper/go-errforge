package errforge

import (
	"fmt"
)

// FromRecover converts a recovered panic value into a *PanicError carrying the
// stack trace of the panic.
//
// It must be called within the deferred function that recovered the panic (or a
// function it calls directly), otherwise the panic stack will already have been
// unwound. FromRecover returns nil when r is nil.
func FromRecover(r interface{}) error {
	if r == nil {
		return nil
	}
	return &PanicError{
		panic:      r,
		stackTrace: capturePanicStack(),
	}
}

// Recover is used to recover from a panic while capturing information about
// where it occurred.
//
// It must be called using "defer". The provided function is invoked only when a
// panic occurred, with a *PanicError describing it.
func Recover(fn func(err error)) {
	if err := FromRecover(recover()); err != nil {
		fn(err)
	}
}

// FromPanic executes fn and returns a *PanicError if fn panics, or nil
// otherwise.
func FromPanic(fn func()) error {
	var err error
	func() {
		defer Recover(func(recoverErr error) {
			err = recoverErr
		})
		fn()
	}()
	return err
}

// PanicError provides the information of a recovered panic.
type PanicError struct {
	panic      interface{}
	stackTrace StackTrace
}

// Panic returns the value provided by the recover() built-in.
func (e *PanicError) Panic() interface{} {
	return e.panic
}

// Stack returns the frames of the stack trace captured when the panic was
// recovered, starting at the site of the panic.
func (e *PanicError) Stack() []*StackFrame {
	return e.stackTrace
}

// StackTrace implements the StackTracer interface.
func (e *PanicError) StackTrace() StackTrace {
	return e.stackTrace
}

// Error implements the error interface.
func (e *PanicError) Error() string {
	return fmt.Sprintf("panic: %v", e.panic)
}

// capturePanicStack captures the current stack and trims everything up to and
// including the runtime's panic dispatcher, so the returned trace starts at the
// function that actually panicked. See https://go.dev/wiki/PanicAndRecover.
func capturePanicStack() StackTrace {
	full := captureStack(1) // skip capturePanicStack itself
	for i, f := range full {
		if f.Package == "runtime" && f.Function == "gopanic" {
			return full[i+1:]
		}
	}
	return full
}
