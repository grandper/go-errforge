package errforge_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/grandper/go-errforge/errforge"
)

func TestToTransient(t *testing.T) {
	t.Run("marks an error as transient without changing its message", func(t *testing.T) {
		err := errforge.ToTransient(errforge.New("service unavailable"))
		assert.Equal(t, "service unavailable", err.Error())
		assert.True(t, errforge.IsTransient(err))
	})

	t.Run("returns nil when the error is nil", func(t *testing.T) {
		assert.NoError(t, errforge.ToTransient(nil))
	})

	t.Run("preserves identity for Is and As", func(t *testing.T) {
		cause := errforge.New("connection refused")
		err := errforge.ToTransient(cause)
		assert.True(t, errforge.Is(err, cause))
		assert.Equal(t, cause, errforge.Unwrap(err))
	})

	t.Run("is discovered anywhere in the chain", func(t *testing.T) {
		err := errforge.ToTransient(errforge.New("boom"))
		err = errforge.Wrap("outer", err)
		err = errforge.Propagate(err, "propagated")
		assert.True(t, errforge.IsTransient(err))
	})

	t.Run("reports false for an untagged error", func(t *testing.T) {
		assert.False(t, errforge.IsTransient(errforge.New("permanent")))
	})

	t.Run("reports false for a nil error", func(t *testing.T) {
		assert.False(t, errforge.IsTransient(nil))
	})
}

func TestIsTransientByCode(t *testing.T) {
	t.Run("reports true for a code that is transient by nature", func(t *testing.T) {
		for _, code := range []errforge.Code{errforge.Unavailable, errforge.Aborted, errforge.DeadlineExceeded} {
			assert.True(t, errforge.IsTransient(errforge.NewWithCode(code, "boom")), "code %s", code)
		}
	})

	t.Run("reports false for a permanent code", func(t *testing.T) {
		assert.False(t, errforge.IsTransient(errforge.NewWithCode(errforge.NotFound, "boom")))
	})

	t.Run("still honors an explicit marker on a permanent code", func(t *testing.T) {
		assert.True(t, errforge.IsTransient(errforge.ToTransient(errforge.NewWithCode(errforge.NotFound, "boom"))))
	})

	t.Run("finds the code under a wrapper", func(t *testing.T) {
		err := errforge.Wrap("outer", errforge.NewWithCode(errforge.Unavailable, "down"))
		assert.True(t, errforge.IsTransient(err))
	})
}
