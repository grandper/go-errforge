package errforge_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grandper/go-errforge/errforge"
)

func TestPublicError(t *testing.T) {
	err := &errforge.PublicError{
		Message:     "unable to connect to your account",
		Reassurance: "your changes were saved",
		Reason:      "we could not connect your account due to a technical issue on our end",
		Resolution:  "please try connecting again",
		WayOut:      "if the issue keeps happening, contact Customer Care",
	}
	const errorMsg = "unable to connect to your account"
	const errorDetails = "Your changes were saved. We could not connect your account due to a technical issue on our end. Please try connecting again. If the issue keeps happening, contact Customer Care"

	t.Run("should provide error description", func(t *testing.T) {
		assert.Equal(t, errorMsg, err.Error())
		assert.Equal(t, errorDetails, err.Details())
	})

	expectedLogStruct := struct {
		Message string `json:"message"`
		Details string `json:"details"`
	}{
		Message: errorMsg,
		Details: errorDetails,
	}

	t.Run("should log data with slog", func(t *testing.T) {
		assertJSONSlog(t, expectedLogStruct, err)
	})

	t.Run("should log data with zap", func(t *testing.T) {
		assertJSONZapLog(t, expectedLogStruct, err)
	})

	t.Run("should log data with zerolog", func(t *testing.T) {
		assertJSONZerolog(t, expectedLogStruct, err)
	})
}

// TestPublicErrorTranslatable asserts that a key and params make the error
// translatable without changing what Error() says, and that both reach the
// logs of the three supported loggers.
func TestPublicErrorTranslatable(t *testing.T) {
	err := &errforge.PublicError{
		Key:     "order.out_of_stock",
		Message: "not enough items in stock",
		Params:  errforge.Params{"requested": 8, "available": 5},
	}

	t.Run("Error still returns the fallback message", func(t *testing.T) {
		assert.Equal(t, "not enough items in stock", err.Error())
		assert.Empty(t, err.Details())
	})

	type logStruct struct {
		Key     string         `json:"key"`
		Message string         `json:"message"`
		Details string         `json:"details"`
		Params  map[string]any `json:"params"`
	}
	expected := logStruct{
		Key:     "order.out_of_stock",
		Message: "not enough items in stock",
		Params:  map[string]any{"requested": float64(8), "available": float64(5)},
	}

	t.Run("logs the key and params with slog", func(t *testing.T) {
		assertJSONSlog(t, expected, err)
	})

	t.Run("logs the key and params with zap", func(t *testing.T) {
		assertJSONZapLog(t, expected, err)
	})

	t.Run("logs the key and params with zerolog", func(t *testing.T) {
		assertJSONZerolog(t, expected, err)
	})

	t.Run("omits the key and params when unset", func(t *testing.T) {
		var buf bytes.Buffer
		slog.New(slog.NewJSONHandler(&buf, nil)).Info("m", "err", &errforge.PublicError{Message: "plain"})

		var entry map[string]any
		require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
		value := entry["err"].(map[string]any)
		assert.NotContains(t, value, "key")
		assert.NotContains(t, value, "params")
	})
}
