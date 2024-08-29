package errforge

import (
	"context"
	stderrors "errors"
	"log/slog"

	"github.com/rs/zerolog"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc/status"
)

// DetailedError is an error enriched with structured, machine-readable
// information: an error code, the location in the source where it was created,
// and an optional user-facing PublicError. It integrates with the slog, zap
// and zerolog loggers to produce a clean structured log entry.
type DetailedError struct {
	// code is a code attached to an error.
	code Code

	// message is a description of the error.
	message string

	// cause is the cause of this error.
	cause error

	// location describes where the error occurred inside the code.
	location *StackFrame

	// publicError contains a human-readable description of the error.
	// In particular the user is provided with hints about the error
	// and how to solve it.
	publicError *PublicError
}

// DetailedOption configures a DetailedError built with Detailed.
type DetailedOption func(*DetailedError)

// WithCode attaches an error code to the error.
func WithCode(code Code) DetailedOption {
	return func(de *DetailedError) { de.code = code }
}

// WithCause sets the underlying cause of the error, making it reachable through
// errors.Unwrap, errors.Is and errors.As.
func WithCause(cause error) DetailedOption {
	return func(de *DetailedError) { de.cause = cause }
}

// WithPublic attaches a user-facing PublicError to the error.
func WithPublic(pe *PublicError) DetailedOption {
	return func(de *DetailedError) { de.publicError = pe }
}

// Detailed creates a DetailedError with the given message. The location of the
// call is captured automatically. Additional information — a code, a cause, or
// a user-facing PublicError — can be supplied through options:
//
//	err := errors.Detailed("query failed",
//	    errors.WithCode(CodeTimeout),
//	    errors.WithCause(cause),
//	)
func Detailed(message string, opts ...DetailedOption) *DetailedError {
	return newDetailed(message, opts...)
}

// newDetailed builds a DetailedError, capturing the location of the user code
// that called the exported constructor. It must be called by exactly one
// exported wrapper (Detailed, Propagate, NewWithCode, ...), so that the frame
// two levels above newDetailed is the user's call site.
func newDetailed(message string, opts ...DetailedOption) *DetailedError {
	de := &DetailedError{
		code:    NoCode,
		message: message,
	}
	// skipToCaller skips newDetailed and its exported wrapper to reach the
	// user's frame.
	const skipToCaller = 2
	if trace := captureStack(skipToCaller); len(trace) > 0 {
		de.location = trace[0]
	}
	for _, opt := range opts {
		opt(de)
	}
	return de
}

// Error returns a description of the error.
func (de *DetailedError) Error() string {
	return de.message
}

// Unwrap returns the error that has been wrapped into the error.
func (de *DetailedError) Unwrap() error {
	return de.cause
}

// Code returns the error code attached to the error. When no code has been set
// explicitly, the code is inherited from the cause; if neither carries a code,
// Code returns NoCode.
func (de *DetailedError) Code() Code {
	if de.code != NoCode {
		return de.code
	}
	return GetCode(de.cause)
}

// ExitCode returns the process exit code associated with the error: 1 when the
// error has no code (NoCode), otherwise the numeric value of the code.
func (de *DetailedError) ExitCode() int {
	code := de.Code()
	if code == NoCode {
		return 1
	}
	return int(code)
}

// Location returns the stack frame describing where the error was created, or
// nil when the location is unknown.
func (de *DetailedError) Location() *StackFrame {
	return de.location
}

// Details returns the user-facing PublicError attached to the error. When none
// has been set explicitly, the PublicError is inherited from the cause, so a
// message attached in the domain layer survives being Propagated on the way
// up; if neither carries one, Details returns nil.
func (de *DetailedError) Details() *PublicError {
	if de.publicError != nil {
		return de.publicError
	}
	return GetDetails(de.cause)
}

// LogValue implements the slog.LogValuer interface to provide a clean log of
// the error when slog is used. The value is the one [Attr] produces, so a
// DetailedError logged as slog.Any("error", de) prints the same thing as
// Attr(de): its whole tree, causes included.
func (de *DetailedError) LogValue() slog.Value {
	return newLogView(de).slogValue()
}

// MarshalLogObject implements the zapcore.ObjectMarshaler interface to provide
// a clean log of the error when zap is used. The object is the one [ZapField]
// produces.
func (de *DetailedError) MarshalLogObject(encoder zapcore.ObjectEncoder) error {
	return newLogView(de).MarshalLogObject(encoder)
}

// MarshalZerologObject implements the zerolog.LogObjectMarshaler interface to
// provide a clean log of the error when zerolog is used. The object is the one
// [LogObject] produces.
func (de *DetailedError) MarshalZerologObject(e *zerolog.Event) {
	newLogView(de).MarshalZerologObject(e)
}

// GetCode returns the error code attached to err.
//
// It reports the code of the first error in err's tree that exposes a
// Code() Code method, walking through wrappers, markers and multierrors. An
// error produced by a gRPC client, which carries a *status.Status, reports the
// equivalent code through [CodeFromGRPC]. A context error is a coded error in
// disguise: context.DeadlineExceeded reports DeadlineExceeded and
// context.Canceled reports Canceled, so ctx.Err() returned as is reaches the
// boundary as a 504 or a 499 rather than a 500. If nothing in the tree carries
// a code, GetCode returns NoCode.
func GetCode(err error) Code {
	var coded interface{ Code() Code }
	if stderrors.As(err, &coded) {
		return coded.Code()
	}
	var st interface{ GRPCStatus() *status.Status }
	if stderrors.As(err, &st) {
		return CodeFromGRPC(st.GRPCStatus().Code())
	}
	if stderrors.Is(err, context.DeadlineExceeded) {
		return DeadlineExceeded
	}
	if stderrors.Is(err, context.Canceled) {
		return Canceled
	}
	return NoCode
}

// GetLocation returns the stack frame describing where err was created.
//
// It reports the location of the first error in err's tree exposing a
// Location() *StackFrame method, or nil when none does.
func GetLocation(err error) *StackFrame {
	var located interface{ Location() *StackFrame }
	if stderrors.As(err, &located) {
		return located.Location()
	}
	return nil
}

// GetDetails returns the user-facing PublicError attached to err.
//
// It reports the details of the first error in err's tree exposing a
// Details() *PublicError method, or nil when none does.
func GetDetails(err error) *PublicError {
	var detailed interface{ Details() *PublicError }
	if stderrors.As(err, &detailed) {
		return detailed.Details()
	}
	return nil
}
