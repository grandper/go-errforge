package errforge

import stderrors "errors"

// ToTransient marks err as transient: an error that may succeed if the failing
// operation is retried (a dropped connection, a timeout, a 503 from a downstream
// service). It returns nil when err is nil, so it can be applied inline on a
// call that may or may not have failed.
//
// The marker preserves err's identity: its Error string is unchanged, it unwraps
// to err, and [Is] / [As] continue to see through it. Read the marker back with
// [IsTransient].
//
// ToTransient decorates an error with a fact about it without changing what it
// is, like [WithStack]. It answers the question "should the caller retry?".
func ToTransient(err error) error {
	if err == nil {
		return nil
	}
	return &transientError{err: err}
}

// IsTransient reports whether err, or any error in its chain, has been marked
// transient with [ToTransient], or carries a [Code] that is transient by nature
// (see [Code.Transient]).
func IsTransient(err error) bool {
	var t interface{ transient() bool }
	if stderrors.As(err, &t) && t.transient() {
		return true
	}
	return GetCode(err).Transient()
}

// transientError tags an error as transient without altering its message.
type transientError struct {
	err error
}

// Error implements the error interface, reporting the wrapped error's message
// unchanged.
func (e *transientError) Error() string { return e.err.Error() }

// Unwrap returns the wrapped error, keeping its identity reachable.
func (e *transientError) Unwrap() error { return e.err }

// transient is the marker read by IsTransient. The method is unexported so the
// tag can only be applied through ToTransient.
func (e *transientError) transient() bool { return true }
