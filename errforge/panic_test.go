package errforge_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grandper/go-errforge/errforge"
)

func callFunctionThatPanics() {
	ts := testStruct{}
	ts.thisWillPanic()
}

type testStruct struct{}

func (ts testStruct) thisWillPanic() {
	panic("a panic occurred")
}

// hasFrame reports whether the stack contains a frame in the given file whose
// function name contains fn.
func hasFrame(stack []*errforge.StackFrame, file, fn string) bool {
	for _, frame := range stack {
		if frame.File == file && strings.Contains(frame.Function, fn) {
			return true
		}
	}
	return false
}

func TestFromRecover(t *testing.T) {
	t.Run("should return nil when there is no panic value", func(t *testing.T) {
		assert.NoError(t, errforge.FromRecover(nil))
	})

	t.Run("should build a PanicError with a stack trace from a recovered panic", func(t *testing.T) {
		defer func() {
			r := recover()
			err := errforge.FromRecover(r)
			require.Error(t, err)

			var panicError *errforge.PanicError
			require.True(t, errforge.As(err, &panicError))
			assert.Equal(t, "panic: a panic occurred", panicError.Error())
			assert.Equal(t, "a panic occurred", panicError.Panic())

			stack := panicError.Stack()
			assert.NotEmpty(t, stack)
			assert.True(t, hasFrame(stack, "panic_test.go", "thisWillPanic"),
				"stack should contain the panicking function, got %+v", stack)
		}()
		callFunctionThatPanics()
	})
}

func TestRecover(t *testing.T) {
	t.Run("should invoke the callback with a PanicError on panic", func(t *testing.T) {
		called := false
		func() {
			defer errforge.Recover(func(err error) {
				called = true
				assert.Equal(t, "panic: a panic occurred", err.Error())
				var panicError *errforge.PanicError
				require.True(t, errforge.As(err, &panicError))
				assert.True(t, hasFrame(panicError.Stack(), "panic_test.go", "thisWillPanic"))
			})
			callFunctionThatPanics()
		}()
		assert.True(t, called, "the recover callback should have been called")
	})

	t.Run("should not invoke the callback when there is no panic", func(t *testing.T) {
		called := false
		func() {
			defer errforge.Recover(func(_ error) {
				called = true
			})
		}()
		assert.False(t, called, "the recover callback must not be called without a panic")
	})
}

func TestFromPanic(t *testing.T) {
	t.Run("should return an error describing the panic", func(t *testing.T) {
		err := errforge.FromPanic(func() {
			callFunctionThatPanics()
		})
		require.Error(t, err)
		assert.Equal(t, "panic: a panic occurred", err.Error())

		var panicError *errforge.PanicError
		require.True(t, errforge.As(err, &panicError))
		assert.True(t, hasFrame(panicError.Stack(), "panic_test.go", "thisWillPanic"))
	})

	t.Run("should return nil when the function does not panic", func(t *testing.T) {
		assert.NoError(t, errforge.FromPanic(func() {}))
	})
}
