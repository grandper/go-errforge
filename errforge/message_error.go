package errforge

import (
	"fmt"
	"io"
)

// New returns a new *Error carrying the provided message.
//
// Each call to New returns a distinct error value even if the message is
// identical, so two errors created with the same message are never equal.
// To create a sentinel error, assign the result to a package-level variable
// and compare against it with [Is]:
//
//	var ErrNotFound = errors.New("not found")
//
// New returns the concrete *[Error] type rather than the bare error interface.
// This is what exposes the [Error.WithErr], [Error.WithDetail] and
// [Error.WithDetailf] builders, which derive an enriched error from a sentinel
// while preserving its identity. *[Error] still satisfies error, so New can be
// used exactly like the standard library's New.
func New(msg string) *Error {
	return &Error{msg: msg}
}

// Newf creates a new error whose message is formatted according to a format
// specifier. It follows the same rules as [fmt.Errorf], including support for
// the %w verb to wrap an existing error.
func Newf(format string, a ...any) error {
	return fmt.Errorf(format, a...)
}

// Error is a simple error holding a message.
//
// Values returned by [New] act as sentinels; the [Error.WithErr],
// [Error.WithDetail] and [Error.WithDetailf] builders derive enriched errors
// from a sentinel without losing its identity under [Is].
type Error struct {
	// msg is the rendered message of this error.
	msg string
	// sentinel is the identity anchor: the original sentinel this error was
	// derived from. It is nil when the value is itself a sentinel.
	sentinel *Error
	// cause is an optional wrapped error attached with WithErr.
	cause error
}

// identity returns the sentinel this error matches under Is: the sentinel it was
// derived from, or itself when it is a sentinel.
func (e *Error) identity() *Error {
	if e.sentinel != nil {
		return e.sentinel
	}
	return e
}

// Error implements the error interface.
func (e *Error) Error() string {
	return e.msg
}

// Format implements the fmt.Formatter interface, writing the error message
// for every verb.
func (e *Error) Format(s fmt.State, _ rune) {
	_, _ = io.WriteString(s, e.Error())
}

// Is reports whether target is the sentinel this error was derived from, so an
// error built with WithErr / WithDetail / WithDetailf still matches its sentinel.
func (e *Error) Is(target error) bool {
	return target == e.identity()
}

// Unwrap returns the cause attached with WithErr, or nil.
func (e *Error) Unwrap() error {
	return e.cause
}

// WithDetail returns a new error whose message is this error's message followed
// by detail ("<message>: <detail>").
//
// The sentinel's identity is preserved: the result still matches the original
// sentinel under [Is]. The receiver is not modified, so a package-level sentinel
// stays a clean, reusable value.
func (e *Error) WithDetail(detail string) *Error {
	return &Error{
		msg:      e.msg + ": " + detail,
		sentinel: e.identity(),
		cause:    e.cause,
	}
}

// WithDetailf is the fmt-style variant of [Error.WithDetail]: it formats detail
// according to a format specifier before appending it.
func (e *Error) WithDetailf(format string, args ...any) *Error {
	return e.WithDetail(fmt.Sprintf(format, args...))
}

// WithErr returns a new error that carries cause while keeping the sentinel's
// identity: the result matches both the original sentinel and cause under [Is],
// and [Unwrap] reaches cause. Both messages are rendered ("<message>: <cause>").
//
// It returns the receiver unchanged when cause is nil, so it can be applied
// inline.
func (e *Error) WithErr(cause error) *Error {
	if cause == nil {
		return e
	}
	return &Error{
		msg:      e.msg + ": " + cause.Error(),
		sentinel: e.identity(),
		cause:    combineCauses(e.cause, cause),
	}
}

// combineCauses keeps both causes reachable when WithErr is applied to an error
// that already carries one, so Is / As can still find either.
func combineCauses(existing, added error) error {
	if existing == nil {
		return added
	}
	return Append(existing, added)
}
