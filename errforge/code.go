package errforge

import (
	"net/http"
	"strconv"

	"google.golang.org/grpc/codes"
)

// Code is the category of an error, independent of the transport used to
// report it. The set of codes mirrors google.rpc.Code, the vocabulary shared by
// gRPC and, through [Code.HTTP], by HTTP: an error is stamped with a Code deep
// in the domain layer and the boundary turns it into the matching status with
// [WriteError] or [GRPCStatus], with no application-defined mapping table.
//
// Attach a code with [WithCode], [NewWithCode] or [PropagateWithCode], and read
// it back anywhere up the stack with [GetCode].
type Code int

// The list of codes. Every value but NoCode has a counterpart in
// google.rpc.Code.
const (
	// NoCode is the zero value, reported by errors that carry no code.
	NoCode Code = iota
	// OK indicates that the request succeeded. It is never attached to an error.
	OK
	// Canceled indicates that the request was canceled, typically by the caller.
	Canceled
	// Unknown indicates an error whose cause is unknown.
	Unknown
	// InvalidArgument indicates that the caller sent an invalid argument.
	InvalidArgument
	// DeadlineExceeded indicates that the deadline expired before the request completed.
	DeadlineExceeded
	// NotFound indicates that a requested entity was not found.
	NotFound
	// AlreadyExists indicates that the entity the caller tried to create already exists.
	AlreadyExists
	// PermissionDenied indicates that the caller is not allowed to perform the request.
	PermissionDenied
	// ResourceExhausted indicates that a resource, such as a quota, has been exhausted.
	ResourceExhausted
	// FailedPrecondition indicates that the system is not in the state required by the request.
	FailedPrecondition
	// Aborted indicates that the request was aborted, typically because of a concurrency conflict.
	Aborted
	// OutOfRange indicates that the request was attempted past the valid range.
	OutOfRange
	// Unimplemented indicates that the request is not implemented or not supported.
	Unimplemented
	// Internal indicates that an internal invariant was broken.
	Internal
	// Unavailable indicates that the service is currently unavailable.
	Unavailable
	// DataLoss indicates unrecoverable data loss or corruption.
	DataLoss
	// Unauthenticated indicates that the request lacks valid authentication credentials.
	Unauthenticated
)

// String returns the name of the code as spelled in google.rpc.Code, such as
// "NOT_FOUND". NoCode is "NO_CODE" and a value outside the list is "CODE(n)".
func (c Code) String() string {
	switch c {
	case NoCode:
		return "NO_CODE"
	case OK:
		return "OK"
	case Canceled:
		return "CANCELLED"
	case Unknown:
		return "UNKNOWN"
	case InvalidArgument:
		return "INVALID_ARGUMENT"
	case DeadlineExceeded:
		return "DEADLINE_EXCEEDED"
	case NotFound:
		return "NOT_FOUND"
	case AlreadyExists:
		return "ALREADY_EXISTS"
	case PermissionDenied:
		return "PERMISSION_DENIED"
	case ResourceExhausted:
		return "RESOURCE_EXHAUSTED"
	case FailedPrecondition:
		return "FAILED_PRECONDITION"
	case Aborted:
		return "ABORTED"
	case OutOfRange:
		return "OUT_OF_RANGE"
	case Unimplemented:
		return "UNIMPLEMENTED"
	case Internal:
		return "INTERNAL"
	case Unavailable:
		return "UNAVAILABLE"
	case DataLoss:
		return "DATA_LOSS"
	case Unauthenticated:
		return "UNAUTHENTICATED"
	default:
		return "CODE(" + strconv.Itoa(int(c)) + ")"
	}
}

// Text returns a sentence describing what the code means, adapted from the
// google.rpc.Code descriptions. It returns the empty string for a value outside
// the list, as http.StatusText does.
func (c Code) Text() string {
	switch c {
	case NoCode:
		return "No code was set."
	case OK:
		return "Not an error; returned on success."
	case Canceled:
		return "The operation was canceled, typically by the caller."
	case Unknown:
		return "Unknown error."
	case InvalidArgument:
		return "The client specified an invalid argument."
	case DeadlineExceeded:
		return "The deadline expired before the operation could complete."
	case NotFound:
		return "Some requested entity was not found."
	case AlreadyExists:
		return "The entity that a client attempted to create already exists."
	case PermissionDenied:
		return "The caller does not have permission to execute the specified operation."
	case ResourceExhausted:
		return "Some resource has been exhausted."
	case FailedPrecondition:
		return "The operation was rejected because the system is not in a state required for its execution."
	case Aborted:
		return "The operation was aborted, typically due to a concurrency issue."
	case OutOfRange:
		return "The operation was attempted past the valid range."
	case Unimplemented:
		return "The operation is not implemented or is not supported in this service."
	case Internal:
		return "Internal error: some invariants expected by the underlying system have been broken."
	case Unavailable:
		return "The service is currently unavailable."
	case DataLoss:
		return "Unrecoverable data loss or corruption."
	case Unauthenticated:
		return "The request does not have valid authentication credentials for the operation."
	default:
		return ""
	}
}

// Transient reports whether an error carrying the code is worth retrying by
// nature: the service was unavailable, the operation was aborted by a
// concurrency conflict, or the deadline expired. [IsTransient] consults it, so
// such errors need no explicit [ToTransient] marker.
func (c Code) Transient() bool {
	switch c { //nolint:exhaustive // only the retryable codes are listed on purpose
	case Unavailable, Aborted, DeadlineExceeded:
		return true
	default:
		return false
	}
}

// Internal reports whether an error carrying the code is, by nature, a failure
// on our side rather than the caller's: a broken invariant (Internal), lost or
// corrupted data (DataLoss), or a failure that could not even be classified
// (Unknown). These are the codes that [Code.HTTP] reports as 500 because of
// what they mean, not for lack of a better status.
//
// Internal and [Code.Transient] are independent: a code is at most one of the
// two, and most codes (NotFound, InvalidArgument, PermissionDenied, ...) are
// neither, since they describe a permanent failure that is the caller's fault.
func (c Code) Internal() bool {
	switch c { //nolint:exhaustive // only the server-side codes are listed on purpose
	case Internal, DataLoss, Unknown:
		return true
	default:
		return false
	}
}

// StatusClientClosedRequest is the HTTP status reported when the client
// canceled the request. It is not part of the HTTP standard but follows the
// nginx convention.
const StatusClientClosedRequest = 499

// HTTP returns the HTTP status matching the code. NoCode and values outside
// the list map to http.StatusInternalServerError.
func (c Code) HTTP() int {
	switch c {
	case NoCode:
		return http.StatusInternalServerError
	case OK:
		return http.StatusOK
	case Canceled:
		return StatusClientClosedRequest
	case Unknown:
		return http.StatusInternalServerError
	case InvalidArgument:
		return http.StatusBadRequest
	case DeadlineExceeded:
		return http.StatusGatewayTimeout
	case NotFound:
		return http.StatusNotFound
	case AlreadyExists:
		return http.StatusConflict
	case PermissionDenied:
		return http.StatusForbidden
	case ResourceExhausted:
		return http.StatusTooManyRequests
	case FailedPrecondition:
		return http.StatusPreconditionFailed
	case Aborted:
		return http.StatusConflict
	case OutOfRange:
		return http.StatusBadRequest
	case Unimplemented:
		return http.StatusNotImplemented
	case Internal:
		return http.StatusInternalServerError
	case Unavailable:
		return http.StatusServiceUnavailable
	case DataLoss:
		return http.StatusInternalServerError
	case Unauthenticated:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

// CodeFromHTTP returns the code matching an HTTP status.
//
// Every status carries a code: the ones with a code of their own are listed,
// then any remaining informational, success or redirection status reports OK,
// any other client error InvalidArgument, and any other server error Internal.
// A value outside the HTTP range maps to Unknown.
//
// Several codes share a status, so the listed statuses report the code that
// [Code.HTTP] maps back to them: 409 Conflict is AlreadyExists, not Aborted,
// and 500 Internal Server Error is Internal, not Unknown or DataLoss.
func CodeFromHTTP(status int) Code {
	switch status {
	case StatusClientClosedRequest:
		return Canceled
	case http.StatusBadRequest:
		return InvalidArgument
	case http.StatusUnauthorized:
		return Unauthenticated
	case http.StatusForbidden:
		return PermissionDenied
	case http.StatusNotFound:
		return NotFound
	case http.StatusMethodNotAllowed, http.StatusNotImplemented:
		return Unimplemented
	case http.StatusRequestTimeout, http.StatusGatewayTimeout:
		return DeadlineExceeded
	case http.StatusConflict:
		return AlreadyExists
	case http.StatusPreconditionFailed, http.StatusPreconditionRequired:
		return FailedPrecondition
	case http.StatusRequestedRangeNotSatisfiable:
		return OutOfRange
	case http.StatusRequestEntityTooLarge, http.StatusTooManyRequests, http.StatusInsufficientStorage:
		return ResourceExhausted
	case http.StatusBadGateway, http.StatusServiceUnavailable:
		return Unavailable
	case http.StatusInternalServerError:
		return Internal
	default:
		return codeFromHTTPClass(status)
	}
}

// httpStatusEnd is the first value past the HTTP status range.
const httpStatusEnd = 600

// codeFromHTTPClass returns the code of the statuses carrying no code of their
// own, from their class.
func codeFromHTTPClass(status int) Code {
	switch {
	case status >= http.StatusContinue && status < http.StatusBadRequest:
		return OK
	case status >= http.StatusBadRequest && status < http.StatusInternalServerError:
		return InvalidArgument
	case status >= http.StatusInternalServerError && status < httpStatusEnd:
		return Internal
	default:
		return Unknown
	}
}

// GRPC returns the gRPC code matching the code. The two lists mirror each
// other but this one starts with NoCode, so the values cannot be converted
// directly: NoCode and values outside the list map to codes.Unknown.
func (c Code) GRPC() codes.Code {
	switch c {
	case NoCode:
		return codes.Unknown
	case OK:
		return codes.OK
	case Canceled:
		return codes.Canceled
	case Unknown:
		return codes.Unknown
	case InvalidArgument:
		return codes.InvalidArgument
	case DeadlineExceeded:
		return codes.DeadlineExceeded
	case NotFound:
		return codes.NotFound
	case AlreadyExists:
		return codes.AlreadyExists
	case PermissionDenied:
		return codes.PermissionDenied
	case ResourceExhausted:
		return codes.ResourceExhausted
	case FailedPrecondition:
		return codes.FailedPrecondition
	case Aborted:
		return codes.Aborted
	case OutOfRange:
		return codes.OutOfRange
	case Unimplemented:
		return codes.Unimplemented
	case Internal:
		return codes.Internal
	case Unavailable:
		return codes.Unavailable
	case DataLoss:
		return codes.DataLoss
	case Unauthenticated:
		return codes.Unauthenticated
	default:
		return codes.Unknown
	}
}

// CodeFromGRPC returns the code matching a gRPC code. Values outside the
// standard list map to Unknown.
func CodeFromGRPC(c codes.Code) Code {
	switch c {
	case codes.OK:
		return OK
	case codes.Canceled:
		return Canceled
	case codes.Unknown:
		return Unknown
	case codes.InvalidArgument:
		return InvalidArgument
	case codes.DeadlineExceeded:
		return DeadlineExceeded
	case codes.NotFound:
		return NotFound
	case codes.AlreadyExists:
		return AlreadyExists
	case codes.PermissionDenied:
		return PermissionDenied
	case codes.ResourceExhausted:
		return ResourceExhausted
	case codes.FailedPrecondition:
		return FailedPrecondition
	case codes.Aborted:
		return Aborted
	case codes.OutOfRange:
		return OutOfRange
	case codes.Unimplemented:
		return Unimplemented
	case codes.Internal:
		return Internal
	case codes.Unavailable:
		return Unavailable
	case codes.DataLoss:
		return DataLoss
	case codes.Unauthenticated:
		return Unauthenticated
	default:
		return Unknown
	}
}
