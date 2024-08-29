// Package errforge provides a richer, drop-in replacement for the standard
// library errors package and fmt.Errorf.
//
// It keeps full compatibility with the Go 1.13 error conventions — every error
// works with the standard [errors.Is], [errors.As] and [errors.Unwrap] — while
// adding a handful of features that make errors easier to build, inspect and
// present:
//
//   - Simple messages and sentinels with [New], enriched at the call site
//     with [Error.WithErr], [Error.WithDetail] and [Error.WithDetailf].
//   - Formatted messages with [Newf].
//   - Wrapping one or many errors, preserving each reference, with [Wrap],
//     [Wrapf] and [Link].
//   - Combining independent errors into a single value with [Append] and
//     [Join]; the result satisfies the [MultiError] interface.
//   - Tying a validation failure to the input it is about with [FieldError],
//     read back with [GetFieldErrors] and listed by [WriteErrorJSON], so a
//     boundary reports every bad field of a request at once.
//   - Matching an error against several sentinels at once with [IsAny].
//   - A transport-neutral [Code] for every error — the google.rpc.Code
//     vocabulary (NotFound, InvalidArgument, ...) — attached with
//     [NewWithCode] or [PropagateWithCode], preserved by [Propagate] and read
//     back with [GetCode].
//   - A [Mapper] that attaches the right code to sentinels you do not own
//     (sql.ErrNoRows, io.EOF, ...) from one table at the adapter edge, with
//     [NewMapper], [Map] and [Mapper.AttachCodeTo].
//   - A classification marker that travels with an error without changing its
//     message or identity: [ToTransient] ("should the caller retry?"), read
//     back with [IsTransient].
//   - Turning recovered panics into errors with [Recover], [FromRecover] and
//     [FromPanic].
//   - User-facing error messages with [PublicError], designed to answer what
//     happened, why, and what to do next, and made translatable by a Key and
//     the raw [Params] the client formats in the user's language.
//   - Turning a coded error into the right transport response, with no
//     mapping table to write, with [WriteError] or [WriteErrorJSON] (HTTP) and
//     [GRPCStatus] (gRPC). Context errors report their code too, so ctx.Err()
//     answers 504 or 499 rather than 500.
//   - Structured logging integration for slog, zap and zerolog.
//   - Small control-flow helpers such as [Handle], [Filter] and [RootCause].
//
// # The name
//
// errforge reads as err + forge. A forge is where raw metal is heated and
// hammered into a finished, purpose-built tool — and that is what this package
// does with a bare error. You start from a plain value and shape it: wrap it,
// attach a code, a captured stack trace, the source location, a classification
// marker, a user-facing message, then map it onto a transport response at the
// boundary. The standard library hands you the raw iron; errforge is the
// workshop where you work it into the error your program actually needs.
//
// Every value the forge produces is still a plain error that obeys the Go 1.13
// conventions, so [errors.Is], [errors.As] and [errors.Unwrap] keep working and
// the package interoperates freely with standard and third-party code.
//
// The package is imported as errforge (the go- prefix lives only on the module
// path, github.com/grandper/go-errforge, matching the sibling go- libraries):
// errforge.New, errforge.Wrap, errforge.WithCode.
package errforge
