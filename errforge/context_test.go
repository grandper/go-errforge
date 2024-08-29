package errforge_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/grandper/go-errforge/errforge"
)

func TestFromContext(t *testing.T) {
	ctx := context.Background()

	t.Run("should return nil when no error occurred", func(t *testing.T) {
		assert.NoError(t, errforge.FromContext(ctx))
	})

	t.Run("should return Cancel error when the context is canceled", func(t *testing.T) {
		canceled, cancel := context.WithCancel(ctx)
		cancel()
		err := errforge.FromContext(canceled)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("should return DeadlineExceeded error when the context timed out", func(t *testing.T) {
		timedOut, cancel := context.WithTimeout(ctx, time.Duration(0))
		defer cancel()
		err := errforge.FromContext(timedOut)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})
}
