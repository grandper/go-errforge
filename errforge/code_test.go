package errforge_test

import (
	"net/http"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"

	"github.com/grandper/go-errforge/errforge"
)

// allCodes lists every code, NoCode included, for the table tests.
func allCodes() []errforge.Code {
	return []errforge.Code{
		errforge.NoCode, errforge.OK, errforge.Canceled, errforge.Unknown, errforge.InvalidArgument,
		errforge.DeadlineExceeded, errforge.NotFound, errforge.AlreadyExists, errforge.PermissionDenied,
		errforge.ResourceExhausted, errforge.FailedPrecondition, errforge.Aborted, errforge.OutOfRange,
		errforge.Unimplemented, errforge.Internal, errforge.Unavailable, errforge.DataLoss,
		errforge.Unauthenticated,
	}
}

func TestCode(t *testing.T) {
	t.Run("defaults to NoCode", func(t *testing.T) {
		var code errforge.Code
		assert.Equal(t, errforge.NoCode, code)
	})
}

func TestCodeString(t *testing.T) {
	t.Run("names every code as google.rpc.Code does", func(t *testing.T) {
		expected := map[errforge.Code]string{
			errforge.NoCode:             "NO_CODE",
			errforge.OK:                 "OK",
			errforge.Canceled:           "CANCELLED",
			errforge.Unknown:            "UNKNOWN",
			errforge.InvalidArgument:    "INVALID_ARGUMENT",
			errforge.DeadlineExceeded:   "DEADLINE_EXCEEDED",
			errforge.NotFound:           "NOT_FOUND",
			errforge.AlreadyExists:      "ALREADY_EXISTS",
			errforge.PermissionDenied:   "PERMISSION_DENIED",
			errforge.ResourceExhausted:  "RESOURCE_EXHAUSTED",
			errforge.FailedPrecondition: "FAILED_PRECONDITION",
			errforge.Aborted:            "ABORTED",
			errforge.OutOfRange:         "OUT_OF_RANGE",
			errforge.Unimplemented:      "UNIMPLEMENTED",
			errforge.Internal:           "INTERNAL",
			errforge.Unavailable:        "UNAVAILABLE",
			errforge.DataLoss:           "DATA_LOSS",
			errforge.Unauthenticated:    "UNAUTHENTICATED",
		}
		for _, code := range allCodes() {
			assert.Equal(t, expected[code], code.String(), "code %d", int(code))
		}
	})

	t.Run("spells out an unknown value", func(t *testing.T) {
		assert.Equal(t, "CODE(42)", errforge.Code(42).String())
	})
}

func TestCodeText(t *testing.T) {
	t.Run("describes what each code means", func(t *testing.T) {
		testCases := []struct {
			code     errforge.Code
			expected string
		}{
			{code: errforge.NoCode, expected: "No code was set."},
			{code: errforge.OK, expected: "Not an error; returned on success."},
			{code: errforge.Canceled, expected: "The operation was canceled, typically by the caller."},
			{code: errforge.Unknown, expected: "Unknown error."},
			{code: errforge.InvalidArgument, expected: "The client specified an invalid argument."},
			{code: errforge.DeadlineExceeded, expected: "The deadline expired before the operation could complete."},
			{code: errforge.NotFound, expected: "Some requested entity was not found."},
			{code: errforge.AlreadyExists, expected: "The entity that a client attempted to create already exists."},
			{
				code:     errforge.PermissionDenied,
				expected: "The caller does not have permission to execute the specified operation.",
			},
			{code: errforge.ResourceExhausted, expected: "Some resource has been exhausted."},
			{
				code:     errforge.FailedPrecondition,
				expected: "The operation was rejected because the system is not in a state required for its execution.",
			},
			{code: errforge.Aborted, expected: "The operation was aborted, typically due to a concurrency issue."},
			{code: errforge.OutOfRange, expected: "The operation was attempted past the valid range."},
			{
				code:     errforge.Unimplemented,
				expected: "The operation is not implemented or is not supported in this service.",
			},
			{
				code:     errforge.Internal,
				expected: "Internal error: some invariants expected by the underlying system have been broken.",
			},
			{code: errforge.Unavailable, expected: "The service is currently unavailable."},
			{code: errforge.DataLoss, expected: "Unrecoverable data loss or corruption."},
			{
				code:     errforge.Unauthenticated,
				expected: "The request does not have valid authentication credentials for the operation.",
			},
		}
		for _, tc := range testCases {
			assert.Equal(t, tc.expected, tc.code.Text(), "code %d", int(tc.code))
		}
	})

	t.Run("returns an empty string for an unknown value", func(t *testing.T) {
		assert.Empty(t, errforge.Code(-1).Text())
	})
}

func TestCodeTransient(t *testing.T) {
	transient := []errforge.Code{errforge.Unavailable, errforge.Aborted, errforge.DeadlineExceeded}
	for _, code := range allCodes() {
		assert.Equal(t, slices.Contains(transient, code), code.Transient(), "code %s", code)
	}
}

func TestCodeInternal(t *testing.T) {
	internal := []errforge.Code{errforge.Internal, errforge.DataLoss, errforge.Unknown}
	for _, code := range allCodes() {
		assert.Equal(t, slices.Contains(internal, code), code.Internal(), "code %s", code)
	}

	t.Run("is never also transient", func(t *testing.T) {
		for _, code := range allCodes() {
			assert.False(t, code.Internal() && code.Transient(), "code %s", code)
		}
	})
}

func TestCodeHTTP(t *testing.T) {
	t.Run("maps every code to its HTTP status", func(t *testing.T) {
		expected := map[errforge.Code]int{
			errforge.NoCode:             http.StatusInternalServerError,
			errforge.OK:                 http.StatusOK,
			errforge.Canceled:           errforge.StatusClientClosedRequest,
			errforge.Unknown:            http.StatusInternalServerError,
			errforge.InvalidArgument:    http.StatusBadRequest,
			errforge.DeadlineExceeded:   http.StatusGatewayTimeout,
			errforge.NotFound:           http.StatusNotFound,
			errforge.AlreadyExists:      http.StatusConflict,
			errforge.PermissionDenied:   http.StatusForbidden,
			errforge.ResourceExhausted:  http.StatusTooManyRequests,
			errforge.FailedPrecondition: http.StatusPreconditionFailed,
			errforge.Aborted:            http.StatusConflict,
			errforge.OutOfRange:         http.StatusBadRequest,
			errforge.Unimplemented:      http.StatusNotImplemented,
			errforge.Internal:           http.StatusInternalServerError,
			errforge.Unavailable:        http.StatusServiceUnavailable,
			errforge.DataLoss:           http.StatusInternalServerError,
			errforge.Unauthenticated:    http.StatusUnauthorized,
		}
		for _, code := range allCodes() {
			assert.Equal(t, expected[code], code.HTTP(), "code %s", code)
		}
	})

	t.Run("maps an unknown value to internal server error", func(t *testing.T) {
		assert.Equal(t, http.StatusInternalServerError, errforge.Code(-1).HTTP())
	})

	t.Run("uses the nginx convention for a client closed request", func(t *testing.T) {
		assert.Equal(t, 499, errforge.StatusClientClosedRequest)
	})
}

func TestCodeFromHTTP(t *testing.T) {
	t.Run("maps every HTTP status to its code", func(t *testing.T) {
		testCases := []struct {
			status   int
			expected errforge.Code
		}{
			{status: http.StatusOK, expected: errforge.OK},
			{status: errforge.StatusClientClosedRequest, expected: errforge.Canceled},
			{status: http.StatusBadRequest, expected: errforge.InvalidArgument},
			{status: http.StatusGatewayTimeout, expected: errforge.DeadlineExceeded},
			{status: http.StatusNotFound, expected: errforge.NotFound},
			{status: http.StatusConflict, expected: errforge.AlreadyExists},
			{status: http.StatusForbidden, expected: errforge.PermissionDenied},
			{status: http.StatusTooManyRequests, expected: errforge.ResourceExhausted},
			{status: http.StatusPreconditionFailed, expected: errforge.FailedPrecondition},
			{status: http.StatusNotImplemented, expected: errforge.Unimplemented},
			{status: http.StatusInternalServerError, expected: errforge.Internal},
			{status: http.StatusServiceUnavailable, expected: errforge.Unavailable},
			{status: http.StatusUnauthorized, expected: errforge.Unauthenticated},
			{status: http.StatusMethodNotAllowed, expected: errforge.Unimplemented},
			{status: http.StatusRequestTimeout, expected: errforge.DeadlineExceeded},
			{status: http.StatusRequestEntityTooLarge, expected: errforge.ResourceExhausted},
			{status: http.StatusRequestedRangeNotSatisfiable, expected: errforge.OutOfRange},
			{status: http.StatusPreconditionRequired, expected: errforge.FailedPrecondition},
			{status: http.StatusBadGateway, expected: errforge.Unavailable},
			{status: http.StatusInsufficientStorage, expected: errforge.ResourceExhausted},
		}
		for _, tc := range testCases {
			assert.Equal(t, tc.expected, errforge.CodeFromHTTP(tc.status), "status %d", tc.status)
		}
	})

	t.Run("reports a success, an informational, or a redirection status as OK", func(t *testing.T) {
		statuses := []int{
			http.StatusContinue, http.StatusOK, http.StatusCreated, http.StatusAccepted,
			http.StatusNoContent, http.StatusMovedPermanently, http.StatusNotModified,
		}
		for _, status := range statuses {
			assert.Equal(t, errforge.OK, errforge.CodeFromHTTP(status), "status %d", status)
		}
	})

	t.Run("maps any other client error to an invalid argument", func(t *testing.T) {
		statuses := []int{http.StatusNotAcceptable, http.StatusGone, http.StatusTeapot, http.StatusUnprocessableEntity}
		for _, status := range statuses {
			assert.Equal(t, errforge.InvalidArgument, errforge.CodeFromHTTP(status), "status %d", status)
		}
	})

	t.Run("maps any other server error to an internal error", func(t *testing.T) {
		statuses := []int{http.StatusHTTPVersionNotSupported, http.StatusLoopDetected, http.StatusNotExtended}
		for _, status := range statuses {
			assert.Equal(t, errforge.Internal, errforge.CodeFromHTTP(status), "status %d", status)
		}
	})

	t.Run("maps a value outside the HTTP range to unknown", func(t *testing.T) {
		for _, status := range []int{0, -1, 42, 600, 999} {
			assert.Equal(t, errforge.Unknown, errforge.CodeFromHTTP(status), "status %d", status)
		}
	})

	t.Run("round-trips the statuses that have a code of their own", func(t *testing.T) {
		statuses := []int{
			http.StatusOK, errforge.StatusClientClosedRequest, http.StatusBadRequest, http.StatusGatewayTimeout,
			http.StatusNotFound, http.StatusConflict, http.StatusForbidden, http.StatusTooManyRequests,
			http.StatusPreconditionFailed, http.StatusNotImplemented, http.StatusInternalServerError,
			http.StatusServiceUnavailable, http.StatusUnauthorized,
		}
		for _, status := range statuses {
			assert.Equal(t, status, errforge.CodeFromHTTP(status).HTTP(), "status %d", status)
		}
	})
}

func TestCodeGRPC(t *testing.T) {
	t.Run("maps every code to its gRPC code", func(t *testing.T) {
		expected := map[errforge.Code]codes.Code{
			errforge.NoCode:             codes.Unknown,
			errforge.OK:                 codes.OK,
			errforge.Canceled:           codes.Canceled,
			errforge.Unknown:            codes.Unknown,
			errforge.InvalidArgument:    codes.InvalidArgument,
			errforge.DeadlineExceeded:   codes.DeadlineExceeded,
			errforge.NotFound:           codes.NotFound,
			errforge.AlreadyExists:      codes.AlreadyExists,
			errforge.PermissionDenied:   codes.PermissionDenied,
			errforge.ResourceExhausted:  codes.ResourceExhausted,
			errforge.FailedPrecondition: codes.FailedPrecondition,
			errforge.Aborted:            codes.Aborted,
			errforge.OutOfRange:         codes.OutOfRange,
			errforge.Unimplemented:      codes.Unimplemented,
			errforge.Internal:           codes.Internal,
			errforge.Unavailable:        codes.Unavailable,
			errforge.DataLoss:           codes.DataLoss,
			errforge.Unauthenticated:    codes.Unauthenticated,
		}
		for _, code := range allCodes() {
			assert.Equal(t, expected[code], code.GRPC(), "code %s", code)
		}
	})

	t.Run("maps an unknown value to Unknown", func(t *testing.T) {
		assert.Equal(t, codes.Unknown, errforge.Code(-1).GRPC())
	})
}

func TestCodeFromGRPC(t *testing.T) {
	t.Run("maps every gRPC code to its code", func(t *testing.T) {
		expected := map[codes.Code]errforge.Code{
			codes.OK:                 errforge.OK,
			codes.Canceled:           errforge.Canceled,
			codes.Unknown:            errforge.Unknown,
			codes.InvalidArgument:    errforge.InvalidArgument,
			codes.DeadlineExceeded:   errforge.DeadlineExceeded,
			codes.NotFound:           errforge.NotFound,
			codes.AlreadyExists:      errforge.AlreadyExists,
			codes.PermissionDenied:   errforge.PermissionDenied,
			codes.ResourceExhausted:  errforge.ResourceExhausted,
			codes.FailedPrecondition: errforge.FailedPrecondition,
			codes.Aborted:            errforge.Aborted,
			codes.OutOfRange:         errforge.OutOfRange,
			codes.Unimplemented:      errforge.Unimplemented,
			codes.Internal:           errforge.Internal,
			codes.Unavailable:        errforge.Unavailable,
			codes.DataLoss:           errforge.DataLoss,
			codes.Unauthenticated:    errforge.Unauthenticated,
			codes.Code(42):           errforge.Unknown,
		}
		for grpcCode, expectedCode := range expected {
			assert.Equal(t, expectedCode, errforge.CodeFromGRPC(grpcCode), "gRPC code %s", grpcCode)
		}
	})

	t.Run("round-trips every code but NoCode", func(t *testing.T) {
		for _, code := range allCodes() {
			if code == errforge.NoCode {
				continue
			}
			assert.Equal(t, code, errforge.CodeFromGRPC(code.GRPC()), "code %s", code)
		}
	})
}
