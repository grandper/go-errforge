package errforge_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grandper/go-errforge/errforge"
)

func TestAppend(t *testing.T) {
	t.Run("return nil when both the error and the list is empty", func(t *testing.T) {
		assert.NoError(t, errforge.Append(nil))
	})

	t.Run("return nil when the error and every appended error are nil", func(t *testing.T) {
		assert.NoError(t, errforge.Append(nil, nil, nil))
	})

	t.Run("return the error when the list is empty", func(t *testing.T) {
		err := errforge.New("hello world")
		assert.Equal(t, err, errforge.Append(err))
	})

	t.Run("return a single error unwrapped when only one is non-nil", func(t *testing.T) {
		err := errforge.New("hello world")
		assert.Equal(t, err, errforge.Append(nil, nil, err))
	})

	t.Run("combine a nil base error with appended errors", func(t *testing.T) {
		err1 := errforge.New("first")
		err2 := errforge.New("second")
		err := errforge.Append(nil, err1, err2)
		assert.Equal(t, "errors occurred: [first, second]", err.Error())
		assert.Equal(t, []error{err1, err2}, errforge.UnwrapErrors(err))
	})

	t.Run("append onto an existing error", func(t *testing.T) {
		base := errforge.New("first")
		err2 := errforge.New("second")
		err := errforge.Append(base, err2)
		assert.Equal(t, "errors occurred: [first, second]", err.Error())
		assert.Equal(t, []error{base, err2}, errforge.UnwrapErrors(err))
	})

	t.Run("append onto an existing MultiError and ignore nil errors", func(t *testing.T) {
		err1 := errforge.New("first")
		err2 := errforge.New("second")
		err3 := errforge.New("third")
		multi := errforge.Append(err1, err2)
		err := errforge.Append(multi, nil, err3, nil)
		assert.Equal(t, "errors occurred: [first, second, third]", err.Error())
		assert.Equal(t, []error{err1, err2, err3}, errforge.UnwrapErrors(err))
	})

	t.Run("result implements the MultiError interface", func(t *testing.T) {
		err1 := errforge.New("first")
		err2 := errforge.New("second")
		err := errforge.Append(err1, err2)
		var multi errforge.MultiError
		ok := errforge.As(err, &multi)
		require.True(t, ok)
		assert.Equal(t, []error{err1, err2}, multi.Errors())
	})

	t.Run("supports errforge.Is and errforge.As over its members", func(t *testing.T) {
		sentinel := errforge.New("sentinel")
		typed := &mainError{}
		err := errforge.Append(sentinel, typed)
		assert.True(t, errforge.Is(err, sentinel))
		var target *mainError
		assert.True(t, errforge.As(err, &target))
	})
}

func TestJoin(t *testing.T) {
	t.Run("errors can be joined", func(t *testing.T) {
		childErr1 := errforge.New("hello world")
		childErr2 := errforge.New("hello univers")
		err := errforge.Join(childErr1, childErr2)
		require.Error(t, err)
		assert.Equal(t, "errors occurred: [hello world, hello univers]", err.Error())

		unwrapedErrs := errforge.UnwrapErrors(err)
		assert.Equal(t, []error{childErr1, childErr2}, unwrapedErrs)
	})

	t.Run("return nil when all errors are nil", func(t *testing.T) {
		assert.NoError(t, errforge.Join(nil, nil, nil))
	})

	t.Run("remove nil when joining error", func(t *testing.T) {
		childErr1 := errforge.New("hello world")
		childErr2 := errforge.New("hello univers")
		err := errforge.Join(nil, childErr1, nil, nil, childErr2, nil)
		require.Error(t, err)
		assert.Equal(t, "errors occurred: [hello world, hello univers]", err.Error())

		unwrapedErrs := errforge.UnwrapErrors(err)
		assert.Equal(t, []error{childErr1, childErr2}, unwrapedErrs)
	})
}

func TestMultiError(t *testing.T) {
	err1 := errforge.New("first")
	err2 := errforge.New("second")

	t.Run("Error renders every member", func(t *testing.T) {
		err := errforge.Join(err1, err2)
		assert.Equal(t, "errors occurred: [first, second]", err.Error())
	})

	t.Run("Errors returns a copy of the members", func(t *testing.T) {
		err := errforge.Join(err1, err2)
		multi := func() errforge.MultiError {
			var target errforge.MultiError
			_ = errforge.As(err, &target)
			return target
		}()
		got := multi.Errors()
		assert.Equal(t, []error{err1, err2}, got)

		// Mutating the returned slice must not affect the error.
		got[0] = errforge.New("mutated")
		assert.Equal(t, []error{err1, err2}, multi.Errors())
	})

	t.Run("Is and As report false when no member matches", func(t *testing.T) {
		err := errforge.Join(err1, err2)
		assert.False(t, errforge.Is(err, errforge.New("absent")))
		var target *mainError
		assert.False(t, errforge.As(err, &target))
	})
}
