package errforge

import "iter"

// walk yields every node of err's tree, depth-first, in the order Print
// follows: the node itself, then its cause (Unwrap() error), then its members
// (Unwrap() []error) each in turn. A node built by Link is followed into the
// error it joins before its cause(s), the same way its Is and As see both.
//
// It is the one place the package walks a tree to gather a fact from every
// node — the opposite of the errors.As getters, which stop at the first match.
// Breaking out of the range loop stops the walk. walk(nil) yields nothing.
func walk(err error) iter.Seq[error] {
	return func(yield func(error) bool) {
		walkNode(err, yield)
	}
}

// walkNode yields err and its subtree to yield and reports whether the walk
// should go on.
func walkNode(err error, yield func(error) bool) bool {
	if err == nil {
		return true
	}
	if !yield(err) {
		return false
	}
	for _, child := range children(err) {
		if !walkNode(child, yield) {
			return false
		}
	}
	return true
}

// children returns the nodes directly below err, in the order walk visits them:
// the error a Link joins first, then the cause (Unwrap() error) or the members
// (Unwrap() []error). It returns nil for a leaf. It is the one definition of
// the tree's shape; walk and the log view both follow it.
func children(err error) []error {
	switch e := err.(type) { //nolint:errorlint // children describes err itself, not anything below it
	case *linkedError:
		return []error{e.err, e.cause}
	case *linkedMultiError:
		return append([]error{e.err}, e.causes...)
	}
	if cause := Unwrap(err); cause != nil {
		return []error{cause}
	}
	return UnwrapErrors(err)
}

// transparentCause returns the cause of err when it repeats err's message — a
// marker such as WithStack or ToTransient, a Propagate that keeps the wording —
// and nil otherwise. Print collapses such a run into one entry, and the log
// view into one object; both step through the run with it.
func transparentCause(err error) error {
	cause := Unwrap(err)
	if cause == nil || cause.Error() != err.Error() {
		return nil
	}
	return cause
}
