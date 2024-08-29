package errforge_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grandper/go-errforge/errforge"
)

func TestMessageError(t *testing.T) {
	t.Run("can be created from a message", func(t *testing.T) {
		err := errforge.New("hello world")
		require.Error(t, err)
		assert.Equal(t, "hello world", err.Error())
	})

	t.Run("can be created from a message with an argument", func(t *testing.T) {
		err := errforge.Newf("hello %s", "world")
		require.Error(t, err)
		assert.Equal(t, "hello world", err.Error())
	})

	t.Run("can be created by wrapping and unwrapping an error", func(t *testing.T) {
		childErr := errforge.New("hello world")
		err := errforge.Newf("child error: %w", childErr)
		require.Error(t, err)
		assert.Equal(t, "child error: hello world", err.Error())

		unwrapedErr := errforge.Unwrap(err)
		assert.Equal(t, childErr, unwrapedErr)
	})

	t.Run("renders its message through the fmt.Formatter interface", func(t *testing.T) {
		err := errforge.New("hello world")
		assert.Equal(t, "hello world", fmt.Sprintf("%s", err))
		assert.Equal(t, "hello world", fmt.Sprintf("%v", err))
	})

	t.Run("two distinct errors with the same message are not equal", func(t *testing.T) {
		assert.False(t, errforge.Is(errforge.New("boom"), errforge.New("boom")))
	})
}

func TestSentinelBuilders(t *testing.T) {
	t.Run("WithDetail renders both messages and keeps the sentinel identity", func(t *testing.T) {
		errNotFound := errforge.New("not found")
		err := errNotFound.WithDetail("user with id 42")

		assert.Equal(t, "not found: user with id 42", err.Error())
		assert.True(t, errforge.Is(err, errNotFound))
	})

	t.Run("WithDetailf formats the detail", func(t *testing.T) {
		errNotFound := errforge.New("not found")
		err := errNotFound.WithDetailf("user with id %d", 42)

		assert.Equal(t, "not found: user with id 42", err.Error())
		assert.True(t, errforge.Is(err, errNotFound))
	})

	t.Run("WithErr renders both messages and matches sentinel and cause", func(t *testing.T) {
		errNotFound := errforge.New("not found")
		cause := errforge.New("connection refused")
		err := errNotFound.WithErr(cause)

		assert.Equal(t, "not found: connection refused", err.Error())
		assert.True(t, errforge.Is(err, errNotFound))
		assert.True(t, errforge.Is(err, cause))
		assert.Equal(t, cause, errforge.Unwrap(err))
	})

	t.Run("WithErr returns the receiver unchanged when the cause is nil", func(t *testing.T) {
		errNotFound := errforge.New("not found")
		assert.Same(t, errNotFound, errNotFound.WithErr(nil))
	})

	t.Run("builders chain a detail and a cause", func(t *testing.T) {
		errNotFound := errforge.New("not found")
		cause := errforge.New("query timed out")
		err := errNotFound.WithDetailf("user with id %d", 42).WithErr(cause)

		assert.Equal(t, "not found: user with id 42: query timed out", err.Error())
		assert.True(t, errforge.Is(err, errNotFound))
		assert.True(t, errforge.Is(err, cause))
	})

	t.Run("chaining two WithErr keeps both causes reachable", func(t *testing.T) {
		errNotFound := errforge.New("not found")
		first := errforge.New("first cause")
		second := errforge.New("second cause")
		err := errNotFound.WithErr(first).WithErr(second)

		assert.True(t, errforge.Is(err, errNotFound))
		assert.True(t, errforge.Is(err, first))
		assert.True(t, errforge.Is(err, second))
	})

	t.Run("the sentinel value itself is not mutated", func(t *testing.T) {
		errNotFound := errforge.New("not found")
		_ = errNotFound.WithDetail("something")
		assert.Equal(t, "not found", errNotFound.Error())
	})

	t.Run("a derived error only matches its own sentinel", func(t *testing.T) {
		errNotFound := errforge.New("not found")
		errOther := errforge.New("other")
		err := errNotFound.WithDetail("x")

		assert.True(t, errforge.Is(err, errNotFound))
		assert.False(t, errforge.Is(err, errOther))
	})

	t.Run("the identity survives standard wrapping", func(t *testing.T) {
		errNotFound := errforge.New("not found")
		err := errforge.Wrap("loading profile", errNotFound.WithDetail("user 42"))
		assert.True(t, errforge.Is(err, errNotFound))
	})
}
