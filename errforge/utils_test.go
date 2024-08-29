package errforge_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/grandper/go-errforge/errforge"
)

func TestHandleAndFilter(t *testing.T) {
	t.Run("Handle returns the value when there is no error", func(t *testing.T) {
		successfulCall := func() (bool, error) { return true, nil }
		assert.True(t, errforge.Handle(successfulCall()))
	})

	t.Run("Handle panics when there is an error", func(t *testing.T) {
		failingCall := func() (bool, error) { return false, errforge.New("an error occurred") }
		assert.Panics(t, func() {
			errforge.Handle(failingCall())
		})
	})

	t.Run("Filter returns the value and discards the error", func(t *testing.T) {
		failingCall := func() (bool, error) { return true, errforge.New("an error occurred") }
		assert.True(t, errforge.Filter(failingCall()))
	})
}

func TestRootCause(t *testing.T) {
	level0Err := errforge.New("an error occurred")
	level1Err := errforge.Newf("a problem happened: %w", level0Err)
	level2Err := errforge.Newf("the application stopped: %w", level1Err)

	t.Run("should return the error itself if there's no internal error", func(t *testing.T) {
		assert.EqualError(t, errforge.RootCause(level0Err), "an error occurred")
	})

	t.Run("should return the wrapped error", func(t *testing.T) {
		assert.EqualError(t, errforge.RootCause(level1Err), "an error occurred")
	})

	t.Run("should return the most internal error", func(t *testing.T) {
		assert.EqualError(t, errforge.RootCause(level2Err), "an error occurred")
	})

	t.Run("should return nil when the error is nil", func(t *testing.T) {
		assert.NoError(t, errforge.RootCause(nil))
	})
}

func TestUnwrap(t *testing.T) {
	t.Run("returns the wrapped error", func(t *testing.T) {
		cause := errforge.New("cause")
		err := errforge.Wrap("context", cause)
		assert.Equal(t, cause, errforge.Unwrap(err))
	})

	t.Run("returns nil when there is nothing to unwrap", func(t *testing.T) {
		assert.NoError(t, errforge.Unwrap(errforge.New("leaf")))
	})
}

func TestUnwrapErrors(t *testing.T) {
	t.Run("returns the wrapped errors", func(t *testing.T) {
		cause1 := errforge.New("cause 1")
		cause2 := errforge.New("cause 2")
		err := errforge.Wrap("context", cause1, cause2)
		assert.Equal(t, []error{cause1, cause2}, errforge.UnwrapErrors(err))
	})

	t.Run("returns nil for a single-wrapped error", func(t *testing.T) {
		err := errforge.Wrap("context", errforge.New("cause"))
		assert.Nil(t, errforge.UnwrapErrors(err))
	})
}

func TestIs(t *testing.T) {
	sentinel := errforge.New("sentinel")
	err := errforge.Wrap("context", sentinel)
	assert.True(t, errforge.Is(err, sentinel))
	assert.False(t, errforge.Is(err, errforge.New("other")))
}

func TestIsAny(t *testing.T) {
	errA := errforge.New("a")
	errB := errforge.New("b")
	errC := errforge.New("c")

	t.Run("matches when any target matches", func(t *testing.T) {
		assert.True(t, errforge.IsAny(errB, errA, errB, errC))
	})

	t.Run("does not match when no target matches", func(t *testing.T) {
		assert.False(t, errforge.IsAny(errforge.New("other"), errA, errB, errC))
	})

	t.Run("returns false when no targets are given", func(t *testing.T) {
		assert.False(t, errforge.IsAny(errA))
	})

	t.Run("matches through a wrapped error", func(t *testing.T) {
		err := errforge.Wrap("context", errB)
		assert.True(t, errforge.IsAny(err, errA, errB))
	})
}

func TestAs(t *testing.T) {
	typed := &mainError{}
	err := errforge.Wrap("context", typed)
	var target *mainError
	assert.True(t, errforge.As(err, &target))
	assert.Equal(t, typed, target)

	var missing *causeError
	assert.False(t, errforge.As(err, &missing))
}

// logStruct and the assertJSON* helpers below are shared by the tests that
// exercise structured logging of PublicError, StackFrame and DetailedError.

type logStruct[T any] struct {
	Time  time.Time `json:"time"`
	Level string    `json:"level"`
	Msg   string    `json:"msg"`
	Value T         `json:"value"`
}

func assertJSONSlog[T any](t *testing.T, expectedLogStruct T, value any) bool {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	logger.Info("foo", "value", value)
	var parsedStruct logStruct[T]
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsedStruct))
	return assert.Equal(t, expectedLogStruct, parsedStruct.Value)
}

func assertJSONZapLog[T any](t *testing.T, expectedLogStruct T, value any) bool {
	var buf bytes.Buffer
	logger := zap.New(
		zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(&buf),
			zapcore.InfoLevel,
		),
	)
	logger.Info("foo", zap.Any("value", value))
	var parsedStruct logStruct[T]
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsedStruct))
	return assert.Equal(t, expectedLogStruct, parsedStruct.Value)
}

func assertJSONZerolog[T any](t *testing.T, expectedLogStruct T, value any) bool {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	logger.Info().Any("value", value).Msg("foo")
	var parsedStruct logStruct[T]
	require.NoError(t, json.Unmarshal(buf.Bytes(), &parsedStruct))
	return assert.Equal(t, expectedLogStruct, parsedStruct.Value)
}
