package errforge_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grandper/go-errforge/errforge"
)

const (
	ecodeTimeout  = errforge.DeadlineExceeded
	ecodeNotFound = errforge.NotFound
)

func TestNewWithCode(t *testing.T) {
	err := errforge.NewWithCode(ecodeNotFound, "user %d not found", 42)
	assert.Equal(t, "user 42 not found", err.Error())
	assert.Equal(t, ecodeNotFound, errforge.GetCode(err))

	loc := errforge.GetLocation(err)
	require.NotNil(t, loc)
	assert.Equal(t, "propagate_test.go", loc.File)
	assert.Equal(t, "TestNewWithCode", loc.Function)
}

func TestPropagate(t *testing.T) {
	t.Run("wraps the cause and captures the location", func(t *testing.T) {
		cause := errforge.New("connection refused")
		err := errforge.Propagate(cause, "dialing %s", "db")
		assert.Equal(t, "dialing db", err.Error())
		assert.True(t, errforge.Is(err, cause))
		assert.Equal(t, cause, errforge.Unwrap(err))

		loc := errforge.GetLocation(err)
		require.NotNil(t, loc)
		assert.Equal(t, "propagate_test.go", loc.File)
	})

	t.Run("preserves the code of the cause", func(t *testing.T) {
		cause := errforge.NewWithCode(ecodeTimeout, "timed out")
		err := errforge.Propagate(cause, "retrying")
		assert.Equal(t, ecodeTimeout, errforge.GetCode(err))
	})

	t.Run("returns nil when the cause is nil", func(t *testing.T) {
		assert.NoError(t, errforge.Propagate(nil, "should be skipped"))
	})
}

func TestPropagateWithCode(t *testing.T) {
	t.Run("overrides the code of the cause", func(t *testing.T) {
		cause := errforge.NewWithCode(ecodeTimeout, "timed out")
		err := errforge.PropagateWithCode(cause, ecodeNotFound, "looking up resource")
		assert.Equal(t, ecodeNotFound, errforge.GetCode(err))
		assert.True(t, errforge.Is(err, cause))
	})

	t.Run("returns nil when the cause is nil", func(t *testing.T) {
		assert.NoError(t, errforge.PropagateWithCode(nil, ecodeNotFound, "skipped"))
	})
}
