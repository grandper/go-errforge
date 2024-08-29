package errforge

import (
	"context"
	"errors"
)

// Mapper maps errors to the code reported to the caller.
// Errors are matched with errors.Is, so wrapped errors map to the code of the error they wrap.
// Mapping an error a second time replaces its code.
// A Mapper is configured once with NewMapper and cannot change afterward.
//
// Use it at an adapter edge, where a dependency returns its own sentinels
// (sql.ErrNoRows, redis.Nil, io.EOF) and the code cannot be attached at the
// origin: [Mapper.Code] answers which code a sentinel maps to, and
// [Mapper.AttachCodeTo] puts that code on the error so it reaches the
// transport boundary through [GetCode].
type Mapper struct {
	mappings map[error]Code
}

// MapperOption configures a Mapper at construction.
type MapperOption func(*Mapper)

// NewMapper creates a Mapper configured with the given options.
//
// context.Canceled and context.DeadlineExceeded are registered by default, to
// Canceled and DeadlineExceeded, as [GetCode] reports them; an option may
// override either.
func NewMapper(opts ...MapperOption) *Mapper {
	mapper := &Mapper{mappings: map[error]Code{
		context.Canceled:         Canceled,
		context.DeadlineExceeded: DeadlineExceeded,
	}}
	for _, opt := range opts {
		opt(mapper)
	}
	return mapper
}

// CodeMapping maps a code to the errors that report it.
type CodeMapping struct {
	code Code
}

// Map starts mapping errors to the code. Complete the mapping with To.
func Map(code Code) *CodeMapping {
	return &CodeMapping{code: code}
}

// To returns an option mapping all the given errors to the code.
// Nil errors are ignored.
func (cm *CodeMapping) To(errs ...error) MapperOption {
	return func(m *Mapper) {
		for _, err := range errs {
			// A nil error would match only a nil error, which always maps to OK.
			if err == nil {
				continue
			}
			m.mappings[err] = cm.code
		}
	}
}

// Code returns the code the error maps to.
// A nil error maps to OK and an error matching no registration maps to Internal.
func (m *Mapper) Code(err error) Code {
	if err == nil {
		return OK
	}
	for target, code := range m.mappings {
		if errors.Is(err, target) {
			return code
		}
	}
	return Internal
}

// AttachCodeTo returns err carrying the code it maps to, so that [GetCode] —
// and through it [WriteError], [GRPCStatus] and [IsTransient] —
// report that code. It returns nil when err is nil, so it can be applied inline
// on a call that may or may not have failed.
//
// A code already in err's chain (attached with [NewWithCode],
// [PropagateWithCode], a gRPC status, or a context error) wins over the
// registry, and err is returned as is: the origin knows better than the edge.
// Otherwise the code is the one [Mapper.Code] returns, Internal when nothing
// matches, so a sentinel the mapper does not know still surfaces as a
// server-side failure rather than as an uncoded error.
//
// The decoration preserves err's identity: its Error string is unchanged, it
// unwraps to err, and [Is] / [As] continue to see through it, like [WithStack]
// and [ToTransient]. When the error is created rather than received,
// [PropagateWithCode] attaches the code and a message in one call.
func (m *Mapper) AttachCodeTo(err error) error {
	if err == nil {
		return nil
	}
	if GetCode(err) != NoCode {
		return err
	}
	return &codedError{err: err, code: m.Code(err)}
}

// codedError attaches a code to an error without altering its message.
type codedError struct {
	err  error
	code Code
}

// Error implements the error interface, reporting the wrapped error's message
// unchanged.
func (e *codedError) Error() string { return e.err.Error() }

// Unwrap returns the wrapped error, keeping its identity reachable.
func (e *codedError) Unwrap() error { return e.err }

// Code returns the attached code, read by GetCode.
func (e *codedError) Code() Code { return e.code }
