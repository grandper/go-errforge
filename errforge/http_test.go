package errforge_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grandper/go-errforge/errforge"
)

func TestWriteError(t *testing.T) {
	t.Run("maps the code to its status and uses the error message", func(t *testing.T) {
		rec := httptest.NewRecorder()
		err := errforge.NewWithCode(errforge.NotFound, "user 42 not found")

		errforge.WriteError(rec, err)

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Equal(t, "user 42 not found", strings.TrimSpace(rec.Body.String()))
	})

	t.Run("uses the PublicError message as the body when present", func(t *testing.T) {
		rec := httptest.NewRecorder()
		err := errforge.Detailed("internal query failed",
			errforge.WithCode(errforge.InvalidArgument),
			errforge.WithPublic(&errforge.PublicError{Message: "please check your input"}),
		)

		errforge.WriteError(rec, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "please check your input", strings.TrimSpace(rec.Body.String()))
	})

	t.Run("falls back to 500 for an error without a code", func(t *testing.T) {
		rec := httptest.NewRecorder()

		errforge.WriteError(rec, errforge.New("boom"))

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Equal(t, "boom", strings.TrimSpace(rec.Body.String()))
	})

	t.Run("surfaces a code set deep in the chain after propagation", func(t *testing.T) {
		rec := httptest.NewRecorder()
		deep := errforge.NewWithCode(errforge.ResourceExhausted, "throttled")
		err := errforge.Propagate(errforge.Propagate(deep, "layer 2"), "layer 1")

		errforge.WriteError(rec, err)

		assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	})

	t.Run("surfaces a code hidden under a marker and a wrapper", func(t *testing.T) {
		rec := httptest.NewRecorder()
		err := errforge.Wrap("loading", errforge.ToTransient(errforge.NewWithCode(errforge.Unavailable, "down")))

		errforge.WriteError(rec, err)

		assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	})

	t.Run("answers a context deadline with 504 and a cancellation with 499", func(t *testing.T) {
		rec := httptest.NewRecorder()
		errforge.WriteError(rec, context.DeadlineExceeded)
		assert.Equal(t, http.StatusGatewayTimeout, rec.Code)

		rec = httptest.NewRecorder()
		errforge.WriteError(rec, errforge.Wrap("loading", context.Canceled))
		assert.Equal(t, errforge.StatusClientClosedRequest, rec.Code)
	})

	t.Run("sets WWW-Authenticate on a 401", func(t *testing.T) {
		rec := httptest.NewRecorder()
		errforge.WriteError(rec, errforge.NewWithCode(errforge.Unauthenticated, "token expired"))
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Equal(t, "token expired", rec.Header().Get("WWW-Authenticate"))
	})

	t.Run("does not set WWW-Authenticate on other statuses", func(t *testing.T) {
		rec := httptest.NewRecorder()
		errforge.WriteError(rec, errforge.NewWithCode(errforge.PermissionDenied, "read only"))
		assert.Empty(t, rec.Header().Get("WWW-Authenticate"))
	})

	t.Run("is a no-op when the error is nil", func(t *testing.T) {
		rec := httptest.NewRecorder()
		errforge.WriteError(rec, nil)
		assert.Equal(t, http.StatusOK, rec.Code) // recorder default, nothing written
		assert.Empty(t, rec.Body.String())
	})
}

// decodeErrorResponse reads the JSON body written by WriteErrorJSON.
func decodeErrorResponse(t *testing.T, rec *httptest.ResponseRecorder) errforge.ErrorResponse {
	t.Helper()
	var body errforge.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	return body
}

func TestWriteErrorJSON(t *testing.T) {
	t.Run("writes the code by name and the message as JSON", func(t *testing.T) {
		rec := httptest.NewRecorder()
		errforge.WriteErrorJSON(rec, errforge.NewWithCode(errforge.NotFound, "track 42 not found"))

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
		assert.JSONEq(t, `{"code":"NOT_FOUND","message":"track 42 not found"}`, rec.Body.String())
	})

	t.Run("uses the PublicError message when present", func(t *testing.T) {
		rec := httptest.NewRecorder()
		err := errforge.Detailed("internal query failed",
			errforge.WithCode(errforge.InvalidArgument),
			errforge.WithPublic(&errforge.PublicError{Message: "please check your input"}),
		)

		errforge.WriteErrorJSON(rec, err)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		body := decodeErrorResponse(t, rec)
		assert.Equal(t, "INVALID_ARGUMENT", body.Code)
		assert.Equal(t, "please check your input", body.Message)
	})

	t.Run("carries the PublicError key and params for translation", func(t *testing.T) {
		rec := httptest.NewRecorder()
		err := errforge.Detailed("quantity exceeds stock",
			errforge.WithCode(errforge.FailedPrecondition),
			errforge.WithPublic(&errforge.PublicError{
				Key:     "order.out_of_stock",
				Message: "not enough items in stock",
				Params:  errforge.Params{"requested": 8, "available": 5},
			}),
		)

		errforge.WriteErrorJSON(rec, err)

		assert.Equal(t, http.StatusPreconditionFailed, rec.Code)
		assert.JSONEq(t, `{
			"code":"FAILED_PRECONDITION",
			"key":"order.out_of_stock",
			"message":"not enough items in stock",
			"params":{"requested":8,"available":5}
		}`, rec.Body.String())
	})

	t.Run("carries the PublicError details when it has some", func(t *testing.T) {
		rec := httptest.NewRecorder()
		err := errforge.Detailed("account service unreachable",
			errforge.WithCode(errforge.Unavailable),
			errforge.WithPublic(&errforge.PublicError{
				Message:     "unable to connect to your account",
				Reassurance: "your changes were saved",
				Resolution:  "please try connecting again",
			}),
		)

		errforge.WriteErrorJSON(rec, err)

		assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
		assert.JSONEq(t, `{
			"code":"UNAVAILABLE",
			"message":"unable to connect to your account",
			"details":"Your changes were saved. Please try connecting again"
		}`, rec.Body.String())
	})

	t.Run("omits key, params and details when the PublicError has none", func(t *testing.T) {
		rec := httptest.NewRecorder()
		err := errforge.Detailed("boom",
			errforge.WithCode(errforge.NotFound),
			errforge.WithPublic(&errforge.PublicError{Message: "no such track"}),
		)

		errforge.WriteErrorJSON(rec, err)

		assert.JSONEq(t, `{"code":"NOT_FOUND","message":"no such track"}`, rec.Body.String())
	})

	t.Run("reports UNKNOWN and 500 for an error without a code", func(t *testing.T) {
		rec := httptest.NewRecorder()
		errforge.WriteErrorJSON(rec, errforge.New("boom"))

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		body := decodeErrorResponse(t, rec)
		assert.Equal(t, "UNKNOWN", body.Code)
		assert.Equal(t, "boom", body.Message)
	})

	t.Run("surfaces a code hidden under a marker and a wrapper", func(t *testing.T) {
		rec := httptest.NewRecorder()
		err := errforge.Wrap("loading", errforge.ToTransient(errforge.NewWithCode(errforge.Unavailable, "down")))

		errforge.WriteErrorJSON(rec, err)

		assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
		assert.Equal(t, "UNAVAILABLE", decodeErrorResponse(t, rec).Code)
	})

	t.Run("answers a context deadline with 504", func(t *testing.T) {
		rec := httptest.NewRecorder()
		errforge.WriteErrorJSON(rec, context.DeadlineExceeded)

		assert.Equal(t, http.StatusGatewayTimeout, rec.Code)
		assert.Equal(t, "DEADLINE_EXCEEDED", decodeErrorResponse(t, rec).Code)
	})

	t.Run("sets WWW-Authenticate on a 401", func(t *testing.T) {
		rec := httptest.NewRecorder()
		errforge.WriteErrorJSON(rec, errforge.NewWithCode(errforge.Unauthenticated, "token expired"))

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Equal(t, "token expired", rec.Header().Get("WWW-Authenticate"))
		assert.JSONEq(t, `{"code":"UNAUTHENTICATED","message":"token expired"}`, rec.Body.String())
	})

	t.Run("is a no-op when the error is nil", func(t *testing.T) {
		rec := httptest.NewRecorder()
		errforge.WriteErrorJSON(rec, nil)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Empty(t, rec.Body.String())
		assert.Empty(t, rec.Header().Get("Content-Type"))
	})
}
