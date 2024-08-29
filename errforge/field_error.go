package errforge

// FieldError ties err to the name of the input it is about — "name", "email",
// "address.city" — so that a boundary can report every failing field of a
// request at once. It returns nil when err is nil, so it can be applied
// inline on a check that may or may not have failed.
//
// The marker preserves err's identity, like [ToTransient]: its Error string is
// unchanged, it unwraps to err, and [Is] / [As] continue to see through it.
// Read the markers back with [GetFieldErrors]; [WriteErrorJSON] lists them in
// the response body.
//
// A field's [Code] and [PublicError] belong to the field: [GetCode] and
// [GetDetails] do not look inside a field error, so a translatable message
// attached to one field never becomes the message of the whole response. The
// request as a whole is described by the error the fields are collected under,
// typically a [PropagateWithCode] with [InvalidArgument].
//
// Nesting marks a field inside a field: FieldError("address", FieldError("city",
// err)) is reported once, as "address.city".
func FieldError(field string, err error) error {
	if err == nil {
		return nil
	}
	return &fieldError{field: field, err: err}
}

// FieldViolation is one failing field reported by [GetFieldErrors]: the name
// given to [FieldError], and the error it wraps.
type FieldViolation struct {
	// Field is the name of the input, with nested names joined by a dot.
	Field string
	// Err is the error attached to the field, with the marker removed, so it
	// still matches the sentinel it was built from under [Is].
	Err error
}

// GetFieldErrors returns every field error in err's tree, in the order they
// were added: the members of a multierror in sequence, a cause after the error
// that wraps it. It returns nil when err is nil or carries no field error.
func GetFieldErrors(err error) []FieldViolation {
	var violations []FieldViolation
	// A run of directly nested field errors is reported as one violation; the
	// walk still yields the inner ones right after the outer one, so they are
	// skipped.
	skip := 0
	for node := range walk(err) {
		fe, ok := node.(*fieldError) //nolint:errorlint // walk visits every node; errors.As would look past this node and double count
		if !ok {
			continue
		}
		if skip > 0 {
			skip--
			continue
		}
		field, inner := fe.field, fe.err
		for {
			nested, isNested := inner.(*fieldError) //nolint:errorlint // only a directly nested field error joins the path
			if !isNested {
				break
			}
			field, inner = field+"."+nested.field, nested.err
			skip++
		}
		violations = append(violations, FieldViolation{Field: field, Err: inner})
	}
	return violations
}

// fieldError tags an error with the name of the input it is about without
// altering its message.
type fieldError struct {
	field string
	err   error
}

// Error implements the error interface, reporting the wrapped error's message
// unchanged.
func (e *fieldError) Error() string { return e.err.Error() }

// Unwrap returns the wrapped error, keeping its identity reachable.
func (e *fieldError) Unwrap() error { return e.err }

// Code stops GetCode at the field: what is coded below belongs to the field,
// not to the request. errors.As returns the first match, so answering NoCode
// here is what keeps it from descending.
func (e *fieldError) Code() Code { return NoCode }

// Details stops GetDetails at the field, for the same reason as Code: a
// PublicError attached to one field must not become the whole response's.
func (e *fieldError) Details() *PublicError { return nil }
