package errforge

import "fmt"

// NewWithCode creates a coded error with a formatted message, capturing the
// location of the call. The message follows the same formatting rules as
// fmt.Sprintf.
func NewWithCode(code Code, format string, args ...any) error {
	return newDetailed(fmt.Sprintf(format, args...), WithCode(code))
}

// Propagate wraps cause with a formatted message and the location of the call,
// preserving the cause's error code. It returns nil when cause is nil, which
// lets callers elide the usual "if err != nil" check:
//
//	return errors.Propagate(process(arg), "failed to process %v", arg)
//
// The message should describe the action that failed. When it would add nothing
// beyond the cause, it may be empty.
func Propagate(cause error, format string, args ...any) error {
	if cause == nil {
		return nil
	}
	return newDetailed(fmt.Sprintf(format, args...), WithCause(cause))
}

// PropagateWithCode is like Propagate but attaches (or overrides) an error code
// instead of inheriting the cause's code. It returns nil when cause is nil.
func PropagateWithCode(cause error, code Code, format string, args ...any) error {
	if cause == nil {
		return nil
	}
	return newDetailed(fmt.Sprintf(format, args...), WithCause(cause), WithCode(code))
}
