package errforge_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grandper/go-errforge/errforge"
)

func TestWithStack(t *testing.T) {
	t.Run("returns nil for a nil error", func(t *testing.T) {
		assert.NoError(t, errforge.WithStack(nil))
	})

	t.Run("preserves the message, identity and cause", func(t *testing.T) {
		cause := errforge.New("boom")
		err := errforge.WithStack(cause)
		assert.Equal(t, "boom", err.Error())
		assert.True(t, errforge.Is(err, cause))
		assert.Equal(t, cause, errforge.Unwrap(err))
	})

	t.Run("captures the call site as the top frame", func(t *testing.T) {
		err := errforge.WithStack(errforge.New("boom"))
		trace := errforge.GetStackTrace(err)
		require.NotEmpty(t, trace)
		assert.Equal(t, "TestWithStack.func3", trace[0].Function)
		assert.Equal(t, "stack_trace_test.go", trace[0].File)
	})
}

func TestGetStackTrace(t *testing.T) {
	t.Run("finds a trace through a wrapped chain", func(t *testing.T) {
		base := errforge.WithStack(errforge.New("boom"))
		wrapped := errforge.Wrap("context", base)
		assert.NotEmpty(t, errforge.GetStackTrace(wrapped))
	})

	t.Run("finds a trace through a multierror", func(t *testing.T) {
		err := errforge.Join(errforge.New("plain"), errforge.WithStack(errforge.New("boom")))
		assert.NotEmpty(t, errforge.GetStackTrace(err))
	})

	t.Run("finds a trace on the joined side of a Link", func(t *testing.T) {
		err := errforge.Link(errforge.WithStack(errforge.New("left")), errforge.New("right"))
		assert.NotEmpty(t, errforge.GetStackTrace(err))
	})

	t.Run("returns the outermost trace", func(t *testing.T) {
		inner := errforge.WithStack(errforge.New("boom"))
		outer := errforge.WithStack(errforge.Wrap("context", inner))
		innerLine := errforge.GetStackTrace(inner)[0].Line
		assert.Equal(t, innerLine+1, errforge.GetStackTrace(outer)[0].Line)
	})

	t.Run("returns nil when no error carries a trace", func(t *testing.T) {
		assert.Nil(t, errforge.GetStackTrace(errforge.New("boom")))
		assert.Nil(t, errforge.GetStackTrace(nil))
	})
}

func TestStackTraceFormat(t *testing.T) {
	trace := errforge.StackTrace{
		{Package: "app", Function: "handle", File: "main.go", Line: 20},
		{Package: "app", Function: "main", File: "main.go", Line: 10},
	}

	t.Run("String renders one frame per line", func(t *testing.T) {
		want := "\tat handle (main.go:20)\n\tat main (main.go:10)\n"
		assert.Equal(t, want, trace.String())
		assert.Equal(t, want, fmt.Sprintf("%s", trace))
	})

	t.Run("the + flag qualifies functions with their package", func(t *testing.T) {
		got := fmt.Sprintf("%+v", trace)
		assert.Contains(t, got, "\tat app.handle (main.go:20)\n", got)
	})
}

func TestPanicErrorStackTrace(t *testing.T) {
	err := errforge.FromPanic(func() {
		callFunctionThatPanics()
	})
	var pe *errforge.PanicError
	require.True(t, errforge.As(err, &pe))

	trace := pe.StackTrace()
	require.NotEmpty(t, trace)
	// The trace starts at the panic site, not at the runtime dispatcher, and
	// the function name keeps its receiver.
	assert.Equal(t, "testStruct.thisWillPanic", trace[0].Function)
	assert.Equal(t, "panic_test.go", trace[0].File)
}
