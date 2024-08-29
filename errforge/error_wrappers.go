package errforge

import (
	stderrors "errors"
	"fmt"
	"regexp"
	"strings"
)

// Wrap creates an error that wraps one or multiple errors.
// If some errors are nil, they will be removed from the error list.
// If all the provided errors are nil, the returned error is nil.
func Wrap(msg string, errs ...error) error {
	if len(errs) == 1 {
		return createWrapError(msg, errs[0])
	}
	filtered := removeNilFromErrors(errs)
	if len(filtered) == 0 {
		return nil
	}
	return &wrapMultiError{
		msg:  msg,
		errs: filtered,
	}
}

// Wrapf wraps one or more errors with a formatted message.
//
// The variadic argument holds, in order, first the values for the format
// verbs found in format, then the error(s) to wrap. The number of leading
// values is determined by counting the verbs in format, so
//
//	errors.Wrapf("reading %s", name, err)
//
// formats "reading <name>" and wraps err. Trailing arguments are expected to be
// errors; nil values and any non-error argument are ignored. If no error
// remains to wrap, Wrapf returns nil.
func Wrapf(format string, argsAndErr ...interface{}) error {
	numVerbs := countVerbs(format)
	args := argsAndErr[:numVerbs]
	errs := convertToErrors(argsAndErr[numVerbs:])
	return Wrap(fmt.Sprintf(format, args...), errs...)
}

func countVerbs(format string) int {
	return len(verbRegex.FindAllString(format, -1))
}

// convertToErrors keeps the error values from args, discarding nil values and
// any argument that is not an error. Non-error arguments are ignored rather
// than causing a panic, so a mistaken call degrades gracefully.
func convertToErrors(args []interface{}) []error {
	result := make([]error, 0, len(args))
	for _, arg := range args {
		if err, ok := arg.(error); ok && err != nil {
			result = append(result, err)
		}
	}
	return result
}

var verbRegex = regexp.MustCompile(
	`%(?P<flag>\#|\+|\-| |0)?((?P<width>[1-9])\.(?P<precision>[1-9])|(?P<widthDefaultPrecision>[1-9])|(?P<widthZeroPrecison>[1-9])\.|\.(?P<precisionDefaultWidth>[1-9]))?(?P<verb>\w{1,9})`,
)

func createWrapError(msg string, err error) error {
	if err == nil {
		return nil
	}
	return &wrapError{
		msg: msg,
		err: err,
	}
}

type wrapError struct {
	msg string
	err error
}

// Error returns a description of the error.
func (e *wrapError) Error() string {
	return e.msg
}

// Unwrap returns the error that has been wrapped into the error.
func (e *wrapError) Unwrap() error {
	return e.err
}

type wrapMultiError struct {
	msg  string
	errs []error
}

// Error returns a description of the error.
func (we *wrapMultiError) Error() string {
	return we.msg
}

// Unwrap returns the errors that have been wrapped into the error.
func (we *wrapMultiError) Unwrap() []error {
	return we.errs
}

// Link connects two errors together.
//
// The Unwrap method will unwrap only err but errors.Is, errors.As works with
// both of the errors.
//
// If causing error is nil, then err is returned.
// If err is nil, then the causing error is returned.
// If both are nil, the function returns nil.
func Link(err error, causes ...error) error {
	if len(causes) == 1 {
		if err == nil {
			return causes[0]
		}
		if causes[0] == nil {
			return err
		}
		return &linkedError{
			err:   err,
			cause: causes[0],
		}
	}
	if err == nil {
		return Join(causes...)
	}
	errs := removeNilFromErrors(causes)
	if len(errs) == 0 {
		return err
	}
	return &linkedMultiError{
		err:    err,
		causes: errs,
	}
}

// linkedError wraps an error with another error.
type linkedError struct {
	err   error
	cause error
}

// Error implements the error interface.
func (le *linkedError) Error() string {
	return fmt.Sprintf("%s: %s", le.err.Error(), le.cause.Error())
}

// Unwrap implements the Wrapper interface.
func (le *linkedError) Unwrap() error {
	return le.cause
}

// Is reports whether any error in err's tree matches target.
func (le *linkedError) Is(target error) bool {
	return stderrors.Is(le.err, target) || stderrors.Is(le.cause, target)
}

// As finds the first error in err's tree that matches target, and if one is found, sets target to that error value and returns true. Otherwise, it returns false.
func (le *linkedError) As(target interface{}) bool {
	return stderrors.As(le.err, target) || stderrors.As(le.cause, target)
}

// linkedMultiError wraps an error with multiple other errors.
type linkedMultiError struct {
	err    error
	causes []error
}

// Error implements the error interface.
func (le *linkedMultiError) Error() string {
	errStrs := make([]string, 0, len(le.causes))
	for _, err := range le.causes {
		errStrs = append(errStrs, err.Error())
	}
	return fmt.Sprintf("%s: [%s]", le.err.Error(), strings.Join(errStrs, ","))
}

// Unwrap implements the Wrapper interface.
func (le *linkedMultiError) Unwrap() []error {
	return le.causes
}

// Is reports whether any error in err's tree matches target.
func (le *linkedMultiError) Is(target error) bool {
	for _, err := range le.causes {
		if stderrors.Is(err, target) {
			return true
		}
	}
	return stderrors.Is(le.err, target)
}

// As finds the first error in err's tree that matches target, and if one is found, sets target to that error value and returns true. Otherwise, it returns false.
func (le *linkedMultiError) As(target interface{}) bool {
	for _, err := range le.causes {
		if stderrors.As(err, target) {
			return true
		}
	}
	return stderrors.As(le.err, target)
}
