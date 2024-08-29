package errforge

import (
	stderrors "errors"
	"strings"
)

// MultiError is an error that contains multiple errors.
type MultiError interface {
	error
	Errors() []error
}

const multiErrorErrorPrefix = "errors occurred: "

// multiError is a slice of errors that can be used as a single error.
type multiError []error

// Error implements the error interface.
func (e multiError) Error() string {
	s := &strings.Builder{}
	s.WriteString(multiErrorErrorPrefix)
	s.WriteString("[")
	for n, err := range e {
		s.WriteString(err.Error())
		if n < len(e)-1 {
			s.WriteString(", ")
		}
	}
	s.WriteString("]")
	return s.String()
}

// Errors implements the MultiError interface.
func (e multiError) Errors() []error {
	s := make([]error, len(e))
	copy(s, e)
	return s
}

// Unwrap implements the Wrapper interface.
func (e multiError) Unwrap() []error {
	return e
}

// As finds the first error in the list that matches target and, if one is
// found, sets target to that error value and returns true.
func (e multiError) As(target interface{}) bool {
	for _, err := range e {
		if stderrors.As(err, target) {
			return true
		}
	}
	return false
}

// Is reports whether any error in the list matches target.
func (e multiError) Is(target error) bool {
	for _, err := range e {
		if stderrors.Is(err, target) {
			return true
		}
	}
	return false
}

// Append appends errs to err, returning a MultiError.
//
// It works similarly to the built-in append function. If err is not already a
// MultiError, one is created. Nil errors are ignored. If, after ignoring nil
// errors, only a single error remains, that error is returned directly rather
// than being wrapped in a MultiError. If no non-nil error remains, Append
// returns nil.
func Append(err error, errs ...error) error {
	if err == nil && len(errs) == 0 {
		return nil
	}
	{
		var errTyp multiError
		switch {
		case stderrors.As(err, &errTyp):
			for _, e := range errs {
				if e != nil {
					errTyp = append(errTyp, e)
				}
			}
			return errTyp
		default:
			var me multiError
			if err != nil {
				me = multiError{err}
			}
			for _, e := range errs {
				if e != nil {
					me = append(me, e)
				}
			}
			if len(me) == 1 {
				return me[0]
			}
			if len(me) == 0 {
				return nil
			}
			return me
		}
	}
}

// Join returns an error that wraps the given errors.
// Any nil error values are discarded.
// Join returns nil if every value in errs is nil.
func Join(errs ...error) error {
	filtered := removeNilFromErrors(errs)
	if len(filtered) == 0 {
		return nil
	}
	return multiError(filtered)
}

func removeNilFromErrors(errs []error) []error {
	n := 0
	for _, err := range errs {
		if err != nil {
			n++
		}
	}
	if n == 0 {
		return nil
	}
	filteredErrs := make([]error, 0, n)
	for _, err := range errs {
		if err != nil {
			filteredErrs = append(filteredErrs, err)
		}
	}
	return filteredErrs
}
