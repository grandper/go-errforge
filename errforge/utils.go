package errforge

import (
	stderrors "errors"
)

// Unwrap returns the result of calling the Unwrap method on err, if err's
// type contains an Unwrap method returning error.
// Otherwise, Unwrap returns nil.
func Unwrap(err error) error {
	return stderrors.Unwrap(err)
}

// UnwrapErrors returns the result of calling the Unwrap method on err, if err's
// type contains an Unwrap method returning []error.
// Otherwise, UnwrapErrors returns nil.
func UnwrapErrors(err error) []error {
	u, ok := err.(interface {
		Unwrap() []error
	})
	if !ok {
		return nil
	}
	return u.Unwrap()
}

// As finds the first error in err's tree that matches target, and if one is found, sets
// target to that error value and returns true. Otherwise, it returns false.
//
// The tree consists of err itself, followed by the errors obtained by repeatedly
// calling its Unwrap() error or Unwrap() []error method. When err wraps multiple
// errors, As examines err followed by a depth-first traversal of its children.
//
// An error matches target if the error's concrete value is assignable to the value
// pointed to by target, or if the error has a method As(interface{}) bool such that
// As(target) returns true. In the latter case, the As method is responsible for
// setting target.
//
// An error type might provide an As method so it can be treated as if it were a
// different error type.
//
// As panics if target is not a non-nil pointer to either a type that implements
// error, or to any interface type.
func As(err error, target any) bool {
	return stderrors.As(err, target)
}

// Is reports whether any error in err's tree matches target.
//
// The tree consists of err itself, followed by the errors obtained by repeatedly calling its Unwrap() error or Unwrap() []error method. When err wraps multiple errors, Is examines err followed by a depth-first traversal of its children.
//
// An error is considered to match a target if it is equal to that target or if it implements a method Is(error) bool such that Is(target) returns true.
//
// An error type might provide an Is method so it can be treated as equivalent to an existing error. For example, if MyError defines
//
// func (m MyError) Is(target error) bool { return target == fs.ErrExist }
// then Is(MyError{}, fs.ErrExist) returns true. See syscall.Errno.Is for an example in the standard library. An Is method should only shallowly compare err and the target and not call [Unwrap] on either.
func Is(err, target error) bool {
	return stderrors.Is(err, target)
}

// IsAny reports whether err matches any of the target errors, i.e. whether
// [Is](err, target) is true for at least one target. It returns false when no
// targets are given.
//
// Each target is compared with the full Is tree walk, so a wrapped or propagated
// error is matched just as reliably as a bare one. IsAny is the variadic
// companion to [Is], useful for collapsing a group of sentinels that should be
// treated identically:
//
//	if errors.IsAny(err, ErrIDEmpty, ErrIDBadFormat, ErrNameTooLong) {
//	    // any validation failure
//	}
func IsAny(err error, targets ...error) bool {
	for _, target := range targets {
		if stderrors.Is(err, target) {
			return true
		}
	}
	return false
}

// RootCause looks for the root cause of the provided error.
// The function will unwrap the error recursively until it
// finds the original error.
func RootCause(err error) error {
	for {
		wrappedErr := Unwrap(err)
		if wrappedErr == nil {
			return err
		}
		err = wrappedErr
	}
}

// Handle returns value if err is nil, and panics with err otherwise.
//
// It is a convenience helper for wrapping a (value, error) returning call when
// a failure is considered unrecoverable, for example during program
// initialization:
//
//	cfg := errors.Handle(loadConfig())
func Handle[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}
	return value
}

// Filter discards the error of a (value, error) returning call and returns only
// the value. It is useful when the error is deliberately ignored.
func Filter[T any](value T, _ error) T {
	return value
}
