package errforge_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/grandper/go-errforge/errforge"
)

func TestDetailedConstructor(t *testing.T) {
	t.Run("captures message, code, cause and location", func(t *testing.T) {
		cause := errforge.New("root cause")
		err := errforge.Detailed("query failed",
			errforge.WithCode(errforge.InvalidArgument),
			errforge.WithCause(cause),
		)

		assert.Equal(t, "query failed", err.Error())
		assert.Equal(t, errforge.InvalidArgument, err.Code())
		assert.Equal(t, cause, errforge.Unwrap(err))
		assert.True(t, errforge.Is(err, cause))

		require.NotNil(t, err.Location())
		assert.Equal(t, "detailed_error_test.go", err.Location().File)
		assert.Equal(t, "TestDetailedConstructor.func1", err.Location().Function)
	})

	t.Run("attaches a public error", func(t *testing.T) {
		public := &errforge.PublicError{Message: "we could not complete your request"}
		err := errforge.Detailed("boom", errforge.WithPublic(public))
		assert.Equal(t, public, err.Details())
		assert.Equal(t, public, errforge.GetDetails(err))
	})

	t.Run("defaults to NoCode", func(t *testing.T) {
		err := errforge.Detailed("boom")
		assert.Equal(t, errforge.NoCode, err.Code())
	})
}

func TestDetailedCodeInheritance(t *testing.T) {
	t.Run("inherits the code of its cause when none is set", func(t *testing.T) {
		cause := errforge.Detailed("inner", errforge.WithCode(errforge.NotFound))
		err := errforge.Detailed("outer", errforge.WithCause(cause))
		assert.Equal(t, errforge.NotFound, err.Code())
		assert.Equal(t, errforge.NotFound, errforge.GetCode(err))
	})

	t.Run("an explicit code overrides the cause", func(t *testing.T) {
		cause := errforge.Detailed("inner", errforge.WithCode(errforge.NotFound))
		err := errforge.Detailed("outer", errforge.WithCause(cause), errforge.WithCode(errforge.Internal))
		assert.Equal(t, errforge.Internal, err.Code())
	})
}

func TestDetailedExitCode(t *testing.T) {
	assert.Equal(t, 1, errforge.Detailed("boom").ExitCode())
	assert.Equal(t, int(errforge.NotFound), errforge.Detailed("boom", errforge.WithCode(errforge.NotFound)).ExitCode())
}

// TestDetailedLoggingConsistency asserts that the three supported loggers emit
// the same structured shape for a DetailedError.
func TestDetailedLoggingConsistency(t *testing.T) {
	makeErr := func() *errforge.DetailedError {
		return errforge.Detailed("query failed",
			errforge.WithCode(errforge.InvalidArgument),
			errforge.WithCause(errforge.New("connection reset")),
			errforge.WithPublic(&errforge.PublicError{Message: "user message", Reason: "a technical issue"}),
		)
	}

	slogValue := func() map[string]any {
		var buf bytes.Buffer
		slog.New(slog.NewJSONHandler(&buf, nil)).Info("m", "err", makeErr())
		var entry map[string]any
		require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
		return entry["err"].(map[string]any)
	}

	zapValue := func() map[string]any {
		var buf bytes.Buffer
		logger := zap.New(zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(&buf), zapcore.InfoLevel,
		))
		logger.Info("m", zap.Object("err", makeErr()))
		var entry map[string]any
		require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
		return entry["err"].(map[string]any)
	}

	zerologValue := func() map[string]any {
		var buf bytes.Buffer
		logger := zerolog.New(&buf)
		logger.Info().Object("err", makeErr()).Msg("m")
		var entry map[string]any
		require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
		return entry["err"].(map[string]any)
	}

	for name, value := range map[string]map[string]any{
		"slog":    slogValue(),
		"zap":     zapValue(),
		"zerolog": zerologValue(),
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, "INVALID_ARGUMENT", value["code"])
			assert.Equal(t, "query failed", value["message"])

			location, ok := value["location"].(map[string]any)
			require.True(t, ok, "location should be a nested object")
			assert.Equal(t, "detailed_error_test.go", location["file"])

			public, ok := value["public_error"].(map[string]any)
			require.True(t, ok, "public_error should be a nested object")
			assert.Equal(t, "user message", public["message"])

			causes, ok := value["causes"].([]any)
			require.True(t, ok, "causes should be an array")
			require.Len(t, causes, 1)
			assert.Equal(t, map[string]any{"message": "connection reset"}, causes[0])
		})
	}
}

// TestDetailedLoggingOmitsNoCode asserts that an uncoded DetailedError logs no
// code key rather than "NO_CODE".
func TestDetailedLoggingOmitsNoCode(t *testing.T) {
	var buf bytes.Buffer
	slog.New(slog.NewJSONHandler(&buf, nil)).Info("m", "err", errforge.Detailed("boom"))

	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	value := entry["err"].(map[string]any)
	assert.NotContains(t, value, "code")
	assert.Equal(t, "boom", value["message"])
}

// TestGettersWalkTheTree asserts that a code, a location or a public error
// stamped deep in an error tree is found from the top, whatever sits above it.
func TestGettersWalkTheTree(t *testing.T) {
	public := &errforge.PublicError{Message: "please retry"}
	coded := errforge.Detailed("down", errforge.WithCode(errforge.Unavailable), errforge.WithPublic(public))

	t.Run("through a ToTransient marker", func(t *testing.T) {
		err := errforge.ToTransient(coded)
		assert.Equal(t, errforge.Unavailable, errforge.GetCode(err))
		assert.Equal(t, coded.Location(), errforge.GetLocation(err))
		assert.Equal(t, public, errforge.GetDetails(err))
	})

	t.Run("through Wrap and WithStack", func(t *testing.T) {
		err := errforge.WithStack(errforge.Wrap("loading", coded))
		assert.Equal(t, errforge.Unavailable, errforge.GetCode(err))
		assert.Equal(t, coded.Location(), errforge.GetLocation(err))
		assert.Equal(t, public, errforge.GetDetails(err))
	})

	t.Run("through Propagate", func(t *testing.T) {
		// Propagate builds a DetailedError with an empty PublicError slot; the
		// slot must not hide the PublicError attached one level down.
		err := errforge.Propagate(errforge.Propagate(coded, "loading"), "handling request")
		assert.Equal(t, errforge.Unavailable, errforge.GetCode(err))
		assert.Equal(t, public, errforge.GetDetails(err))
	})

	t.Run("a PublicError set on the Propagate wins over the cause's", func(t *testing.T) {
		own := &errforge.PublicError{Message: "own message"}
		err := errforge.Detailed("loading", errforge.WithCause(coded), errforge.WithPublic(own))
		assert.Equal(t, own, errforge.GetDetails(err))
	})

	t.Run("through a multierror", func(t *testing.T) {
		err := errforge.Join(errforge.New("first"), coded)
		assert.Equal(t, errforge.Unavailable, errforge.GetCode(err))
		assert.Equal(t, public, errforge.GetDetails(err))
	})

	t.Run("reads the code of an error received from a gRPC client", func(t *testing.T) {
		downstream := status.Error(codes.NotFound, "no such track")
		assert.Equal(t, errforge.NotFound, errforge.GetCode(downstream))
		assert.Equal(t, errforge.NotFound, errforge.GetCode(errforge.Wrap("loading track", downstream)))
	})

	t.Run("prefers a code set by errforge over a downstream gRPC status", func(t *testing.T) {
		downstream := status.Error(codes.NotFound, "no such track")
		err := errforge.PropagateWithCode(downstream, errforge.FailedPrecondition, "track required")
		assert.Equal(t, errforge.FailedPrecondition, errforge.GetCode(err))
	})

	t.Run("reads a context deadline as DeadlineExceeded", func(t *testing.T) {
		assert.Equal(t, errforge.DeadlineExceeded, errforge.GetCode(context.DeadlineExceeded))
		assert.Equal(t, errforge.DeadlineExceeded, errforge.GetCode(errforge.Wrap("loading", context.DeadlineExceeded)))
	})

	t.Run("reads a context cancellation as Canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		assert.Equal(t, errforge.Canceled, errforge.GetCode(ctx.Err()))
		assert.Equal(t, errforge.Canceled, errforge.GetCode(errforge.Propagate(ctx.Err(), "loading")))
	})

	t.Run("prefers a code set by errforge over a context error", func(t *testing.T) {
		err := errforge.PropagateWithCode(context.Canceled, errforge.Aborted, "gave up")
		assert.Equal(t, errforge.Aborted, errforge.GetCode(err))
	})
}

func newTestDetailedError() *errforge.DetailedError {
	return errforge.Detailed("something went wrong",
		errforge.WithCode(errforge.NotFound),
		errforge.WithCause(errforge.New("root cause")),
		errforge.WithPublic(&errforge.PublicError{
			Message: "we could not complete your request",
			Reason:  "a technical issue occurred",
		}),
	)
}

func TestDetailedErrorAccessors(t *testing.T) {
	de := newTestDetailedError()

	assert.Equal(t, "something went wrong", de.Error())
	assert.Equal(t, "root cause", de.Unwrap().Error())
	assert.Equal(t, errforge.NotFound, de.Code())
	require.NotNil(t, de.Location())
	assert.Equal(t, "detailed_error_test.go", de.Location().File)
	require.NotNil(t, de.Details())
	assert.Equal(t, "we could not complete your request", de.Details().Message)
}

func TestDetailedErrorGetters(t *testing.T) {
	de := newTestDetailedError()

	assert.Equal(t, errforge.NotFound, errforge.GetCode(de))
	assert.Equal(t, de.Location(), errforge.GetLocation(de))
	assert.Equal(t, de.Details(), errforge.GetDetails(de))

	// The getters fall back to sane defaults for plain errors.
	plain := errforge.New("plain")
	assert.Equal(t, errforge.NoCode, errforge.GetCode(plain))
	assert.Nil(t, errforge.GetLocation(plain))
	assert.Nil(t, errforge.GetDetails(plain))
	assert.Equal(t, errforge.NoCode, errforge.GetCode(nil))
}

func TestDetailedErrorSlog(t *testing.T) {
	de := errforge.Detailed("boom", errforge.WithCode(errforge.Unavailable))

	var buf bytes.Buffer
	slog.New(slog.NewJSONHandler(&buf, nil)).Info("m", "err", de)

	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	value, ok := entry["err"].(map[string]any)
	require.True(t, ok, "expected err to be logged as a group")
	assert.Equal(t, "UNAVAILABLE", value["code"])
	assert.Equal(t, "boom", value["message"])
}

func TestDetailedErrorZap(t *testing.T) {
	de := newTestDetailedError()

	var buf bytes.Buffer
	logger := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&buf),
		zapcore.InfoLevel,
	))
	logger.Info("m", zap.Object("err", de))

	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	value := entry["err"].(map[string]any)
	assert.Equal(t, "NOT_FOUND", value["code"])
	assert.Equal(t, "something went wrong", value["message"])
}

func TestDetailedErrorZerolog(t *testing.T) {
	de := newTestDetailedError()

	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	logger.Info().Object("err", de).Msg("m")

	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	value := entry["err"].(map[string]any)
	assert.Equal(t, "NOT_FOUND", value["code"])
	assert.Equal(t, "something went wrong", value["message"])
}
