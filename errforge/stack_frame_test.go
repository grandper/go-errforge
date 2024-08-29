package errforge_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grandper/go-errforge/errforge"
)

//nolint:gochecknoglobals // package-level so CreateStackFrame records the init frame under test
var testStackFrame = errforge.CreateStackFrame()

func TestStackFrameOutsideFunc(t *testing.T) {
	assert.NotNil(t, testStackFrame)
	assert.Equal(t, "stack_frame_test.go", testStackFrame.File)
	assert.Equal(t, "errforge_test", testStackFrame.Package)
	assert.Equal(t, "init", testStackFrame.Function)
	assert.Equal(t, 14, testStackFrame.Line)
	assert.Equal(t, "github.com/grandper/go-errforge/errforge/stack_frame_test.go:14", testStackFrame.String())
}

func TestStackFrameInFunc(t *testing.T) {
	stackTrace := errforge.CreateStackFrame()
	assert.NotNil(t, stackTrace)
	assert.Equal(t, "stack_frame_test.go", stackTrace.File)
	assert.Equal(t, "errforge_test", stackTrace.Package)
	assert.Equal(t, "TestStackFrameInFunc", stackTrace.Function)
	assert.Equal(t, 26, stackTrace.Line)
	assert.Equal(t, "github.com/grandper/go-errforge/errforge/stack_frame_test.go:26", stackTrace.String())
}

func TestStackFrameLogging(t *testing.T) {
	stackTrace := errforge.CreateStackFrame()
	expectedLogStruct := struct {
		Path     string `json:"path"`
		Package  string `json:"package"`
		Function string `json:"function"`
		File     string `json:"file"`
		Line     int    `json:"line"`
	}{
		Path:     "github.com/grandper/go-errforge/errforge",
		Package:  "errforge_test",
		Function: "TestStackFrameLogging",
		File:     "stack_frame_test.go",
		Line:     36,
	}

	t.Run("should log data with slog", func(t *testing.T) {
		assertJSONSlog(t, expectedLogStruct, stackTrace)
	})

	t.Run("should log data with zap", func(t *testing.T) {
		assertJSONZapLog(t, expectedLogStruct, stackTrace)
	})

	t.Run("should log data with zerolog", func(t *testing.T) {
		assertJSONZerolog(t, expectedLogStruct, stackTrace)
	})
}

func TestStackFrameFormat(t *testing.T) {
	sf := &errforge.StackFrame{
		Path:     "github.com/grandper/go-errforge/errforge",
		Package:  "errforge",
		Function: "Foo",
		File:     "foo.go",
		Line:     42,
	}

	tests := []struct {
		format string
		want   string
	}{
		{"%h", "github.com/grandper/go-errforge/errforge"},
		{"%f", "foo.go"},
		{"%n", "Foo"},
		{"%+n", "errforge.Foo"},
		{"%d", "42"},
		{"%s", "foo.go"},
		{"%+s", "Foo\n\tfoo.go"},
		{"%v", "foo.go"},
		{"%+v", "Foo\n\tfoo.go:42"},
	}
	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			assert.Equal(t, tt.want, fmt.Sprintf(tt.format, sf))
		})
	}
}

func TestStackFrameMarshalText(t *testing.T) {
	t.Run("formats function, file and line", func(t *testing.T) {
		sf := &errforge.StackFrame{Function: "Foo", File: "foo.go", Line: 42}
		text, err := sf.MarshalText()
		require.NoError(t, err)
		assert.Equal(t, "Foo foo.go:42", string(text))
	})

	t.Run("returns the name verbatim when the function is unknown", func(t *testing.T) {
		sf := &errforge.StackFrame{Function: "unknown"}
		text, err := sf.MarshalText()
		require.NoError(t, err)
		assert.Equal(t, "unknown", string(text))
	})
}
