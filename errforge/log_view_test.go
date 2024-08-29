package errforge_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/grandper/go-errforge/errforge"
)

// logWith renders err through the three loggers and returns, per logger, the
// decoded "error" value of the line.
func logWith(t *testing.T, err error) map[string]any {
	t.Helper()
	decode := func(line []byte) any {
		var entry map[string]any
		require.NoError(t, json.Unmarshal(line, &entry))
		return entry["error"]
	}

	var slogBuf bytes.Buffer
	slog.New(slog.NewJSONHandler(&slogBuf, nil)).Error("m", errforge.Attr(err))

	var zapBuf bytes.Buffer
	zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&zapBuf), zapcore.InfoLevel,
	)).Error("m", errforge.ZapField(err))

	var zerologBuf bytes.Buffer
	zerologLogger := zerolog.New(&zerologBuf)
	zerologLogger.Error().Object("error", errforge.LogObject(err)).Msg("m")

	return map[string]any{
		"slog":    decode(slogBuf.Bytes()),
		"zap":     decode(zapBuf.Bytes()),
		"zerolog": decode(zerologBuf.Bytes()),
	}
}

// object asserts that value is a JSON object and returns it.
func object(t *testing.T, value any) map[string]any {
	t.Helper()
	obj, ok := value.(map[string]any)
	require.True(t, ok, "expected an object, got %T", value)
	return obj
}

// causes asserts that the object has exactly n causes and returns them.
func causes(t *testing.T, obj map[string]any, n int) []map[string]any {
	t.Helper()
	list, ok := obj["causes"].([]any)
	require.True(t, ok, "expected causes to be an array, got %T", obj["causes"])
	require.Len(t, list, n)
	out := make([]map[string]any, 0, n)
	for _, item := range list {
		out = append(out, object(t, item))
	}
	return out
}

// forEachLogger runs check on the rendering of err by each logger.
func forEachLogger(t *testing.T, err error, check func(t *testing.T, obj map[string]any)) {
	t.Helper()
	for name, value := range logWith(t, err) {
		t.Run(name, func(t *testing.T) {
			check(t, object(t, value))
		})
	}
}

// TestLogViewMarkerOnTop is the motivating case: the marker on top of the tree
// used to hide everything below it from the loggers.
func TestLogViewMarkerOnTop(t *testing.T) {
	err := errforge.ToTransient(
		errforge.PropagateWithCode(context.DeadlineExceeded, errforge.DeadlineExceeded, "calling payment service"),
	)

	forEachLogger(t, err, func(t *testing.T, obj map[string]any) {
		assert.Equal(t, "calling payment service", obj["message"])
		assert.Equal(t, "DEADLINE_EXCEEDED", obj["code"])
		assert.Equal(t, true, obj["transient"])

		location := object(t, obj["location"])
		assert.Equal(t, "log_view_test.go", location["file"])
		assert.Equal(t, "errforge_test", location["package"])
		assert.Equal(t, "TestLogViewMarkerOnTop", location["function"])

		// The marker and the Propagate share the message: one entry, not two.
		cause := causes(t, obj, 1)[0]
		assert.Equal(t, "context deadline exceeded", cause["message"])
		assert.Equal(t, "DEADLINE_EXCEEDED", cause["code"])
		assert.NotContains(t, cause, "location")
		assert.NotContains(t, cause, "causes")
	})
}

func TestLogViewOmitsEmptyKeys(t *testing.T) {
	forEachLogger(t, errforge.New("boom"), func(t *testing.T, obj map[string]any) {
		assert.Equal(t, map[string]any{"message": "boom"}, obj)
	})
}

func TestLogViewStackUnderTheNodeThatCarriesIt(t *testing.T) {
	err := errforge.Wrap("query failed", errforge.WithStack(errforge.New("boom")))

	forEachLogger(t, err, func(t *testing.T, obj map[string]any) {
		assert.Equal(t, "query failed", obj["message"])
		assert.NotContains(t, obj, "stack")

		cause := causes(t, obj, 1)[0]
		assert.Equal(t, "boom", cause["message"])
		stack, ok := cause["stack"].([]any)
		require.True(t, ok, "expected stack to be an array, got %T", cause["stack"])
		require.NotEmpty(t, stack)
		first, ok := stack[0].(string)
		require.True(t, ok)
		assert.True(
			t,
			strings.HasPrefix(first, "errforge_test.TestLogViewStackUnderTheNodeThatCarriesIt (log_view_test.go:"),
			first,
		)
	})
}

func TestLogViewCollapsesARunOfMarkers(t *testing.T) {
	cause := errforge.New("dial tcp: refused")
	err := errforge.ToTransient(errforge.WithStack(errforge.Propagate(cause, "calling payment service")))

	forEachLogger(t, err, func(t *testing.T, obj map[string]any) {
		// The three same-message nodes are one entry holding all their facts.
		assert.Equal(t, "calling payment service", obj["message"])
		assert.Equal(t, true, obj["transient"])
		assert.Contains(t, obj, "stack")
		assert.Contains(t, obj, "location")
		assert.NotContains(t, obj, "code")

		got := causes(t, obj, 1)[0]
		assert.Equal(t, map[string]any{"message": "dial tcp: refused"}, got)
	})
}

func TestLogViewInheritsCodeAndPublicError(t *testing.T) {
	public := &errforge.PublicError{Message: "please retry", Reason: "the service is busy"}
	down := errforge.Detailed("down", errforge.WithCode(errforge.Unavailable), errforge.WithPublic(public))
	err := errforge.Wrap("loading", down)

	forEachLogger(t, err, func(t *testing.T, obj map[string]any) {
		// Wrap carries neither, yet the entry answers what GetCode and GetDetails
		// answer when handed the Wrap.
		assert.Equal(t, "UNAVAILABLE", obj["code"])
		assert.Equal(t, true, obj["transient"], "Unavailable is transient by nature")
		assert.NotContains(t, obj, "location", "the location belongs to the node that carries it")
		pe := object(t, obj["public_error"])
		assert.Equal(t, "please retry", pe["message"])
		assert.Equal(t, "The service is busy", pe["details"])

		cause := causes(t, obj, 1)[0]
		assert.Equal(t, "down", cause["message"])
		assert.Equal(t, "UNAVAILABLE", cause["code"])
		assert.Contains(t, cause, "location")
		assert.Equal(t, pe, object(t, cause["public_error"]))
	})
}

func TestLogViewFieldErrors(t *testing.T) {
	errEmpty := errforge.New("must not be empty")
	err := errforge.PropagateWithCode(
		errforge.Join(
			errforge.FieldError("name", errEmpty),
			errforge.FieldError("address", errforge.FieldError("city", errEmpty)),
			errforge.FieldError("email", errforge.NewWithCode(errforge.InvalidArgument, "not an address")),
		),
		errforge.InvalidArgument, "invalid request",
	)

	forEachLogger(t, err, func(t *testing.T, obj map[string]any) {
		assert.Equal(t, "INVALID_ARGUMENT", obj["code"])

		multi := causes(t, obj, 1)[0]
		assert.Equal(t, "errors occurred: [must not be empty, must not be empty, not an address]", multi["message"])
		assert.NotContains(t, multi, "code", "a code inside a field belongs to the field, not to the request")
		assert.NotContains(t, multi, "field")

		fields := causes(t, multi, 3)
		assert.Equal(t, map[string]any{"message": "must not be empty", "field": "name"}, fields[0])
		assert.Equal(t, map[string]any{"message": "must not be empty", "field": "address.city"}, fields[1])
		assert.Equal(t, "email", fields[2]["field"])
		assert.Equal(t, "INVALID_ARGUMENT", fields[2]["code"], "the field entry reports the field's own code")
		assert.Contains(t, fields[2], "location")
		for _, field := range fields {
			assert.NotContains(t, field, "causes", "the marker and the error it tags are one entry")
		}
	})
}

func TestLogViewMultierrorMembersAreCauses(t *testing.T) {
	err := errforge.Wrap("invalid user", errforge.New("name is empty"), errforge.New("email is invalid"))

	forEachLogger(t, err, func(t *testing.T, obj map[string]any) {
		assert.Equal(t, "invalid user", obj["message"])
		members := causes(t, obj, 2)
		assert.Equal(t, "name is empty", members[0]["message"])
		assert.Equal(t, "email is invalid", members[1]["message"])
	})
}

func TestLogViewLinkListsBothSides(t *testing.T) {
	err := errforge.Link(errforge.New("request failed"), errforge.New("timeout"))

	forEachLogger(t, err, func(t *testing.T, obj map[string]any) {
		assert.Equal(t, "request failed: timeout", obj["message"])
		sides := causes(t, obj, 2)
		assert.Equal(t, "request failed", sides[0]["message"])
		assert.Equal(t, "timeout", sides[1]["message"])
	})
}

func TestLogViewForeignErrors(t *testing.T) {
	t.Run("a fmt.Errorf chain", func(t *testing.T) {
		err := errforge.Newf("reading config: %w", errforge.New("eof"))
		forEachLogger(t, err, func(t *testing.T, obj map[string]any) {
			assert.Equal(t, "reading config: eof", obj["message"])
			assert.Equal(t, map[string]any{"message": "eof"}, causes(t, obj, 1)[0])
		})
	})

	t.Run("a context error reports its code", func(t *testing.T) {
		forEachLogger(t, context.Canceled, func(t *testing.T, obj map[string]any) {
			assert.Equal(t, map[string]any{"message": "context canceled", "code": "CANCELLED"}, obj)
		})
	})

	t.Run("a recovered panic logs its stack", func(t *testing.T) {
		err := errforge.FromPanic(func() { panic("blew up") })
		forEachLogger(t, err, func(t *testing.T, obj map[string]any) {
			assert.Equal(t, "panic: blew up", obj["message"])
			assert.Contains(t, obj, "stack")
		})
	})
}

func TestLogViewNil(t *testing.T) {
	t.Run("slog drops the empty attribute", func(t *testing.T) {
		assert.Equal(t, slog.Attr{}, errforge.Attr(nil))
		var buf bytes.Buffer
		slog.New(slog.NewJSONHandler(&buf, nil)).Error("m", errforge.Attr(nil))
		assert.NotContains(t, buf.String(), "error")
	})

	t.Run("zap skips the field", func(t *testing.T) {
		assert.Equal(t, zap.Skip(), errforge.ZapField(nil))
		var buf bytes.Buffer
		zap.New(zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(&buf), zapcore.InfoLevel,
		)).Error("m", errforge.ZapField(nil))
		assert.NotContains(t, buf.String(), `"error":`)
	})

	t.Run("zerolog logs null", func(t *testing.T) {
		assert.Nil(t, errforge.LogObject(nil))
		var buf bytes.Buffer
		logger := zerolog.New(&buf)
		logger.Error().Object("error", errforge.LogObject(nil)).Msg("m")
		assert.Contains(t, buf.String(), `"error":null`)
	})
}

// TestLogViewDetailedErrorMatchesAttr asserts that a DetailedError logged the
// old way, as a value, prints exactly what Attr prints.
func TestLogViewDetailedErrorMatchesAttr(t *testing.T) {
	de := errforge.Detailed("query failed",
		errforge.WithCode(errforge.InvalidArgument),
		errforge.WithCause(errforge.ToTransient(errforge.New("connection reset"))),
		errforge.WithPublic(&errforge.PublicError{Message: "user message"}),
	)

	render := func(attr slog.Attr) string {
		var buf bytes.Buffer
		slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
			ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
				if a.Key == slog.TimeKey {
					return slog.Attr{}
				}
				return a
			},
		})).Error("m", attr)
		return buf.String()
	}

	viaValue := render(slog.Any("error", de))
	viaAttr := render(errforge.Attr(de))
	assert.Equal(t, viaAttr, viaValue)
	assert.Contains(t, viaValue, `"causes":[{"message":"connection reset","transient":true}]`)
}

// TestLogViewSlogTextHandler checks that the text handler, which cannot nest
// objects in an array, still prints the causes legibly.
func TestLogViewSlogTextHandler(t *testing.T) {
	err := errforge.Wrap("loading", errforge.New("boom"))
	var buf bytes.Buffer
	slog.New(slog.NewTextHandler(&buf, nil)).Error("m", errforge.Attr(err))
	assert.Contains(t, buf.String(), `error.message=loading`)
	assert.Contains(t, buf.String(), `error.causes="[{\"message\":\"boom\"}]"`)
}

// TestLogViewRekey shows how to log under another key than "error".
func TestLogViewRekey(t *testing.T) {
	err := errforge.New("boom")

	attr := errforge.Attr(err)
	attr.Key = "err"
	var buf bytes.Buffer
	slog.New(slog.NewJSONHandler(&buf, nil)).Error("m", attr)
	assert.Contains(t, buf.String(), `"err":{"message":"boom"}`)

	field := errforge.ZapField(err)
	field.Key = "err"
	assert.Equal(t, "err", field.Key)
}
