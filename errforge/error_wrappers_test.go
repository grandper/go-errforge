package errforge_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grandper/go-errforge/errforge"
)

func TestWrappedErrors(t *testing.T) {
	t.Run("can be wrapped in another error and unwrapped", func(t *testing.T) {
		childErr := errforge.New("hello world")
		require.NoError(t, errforge.Unwrap(childErr))

		err := errforge.Wrap("error message", childErr)
		require.Error(t, err)
		assert.Equal(t, "error message", err.Error())

		unwrapedErr := errforge.Unwrap(err)
		assert.Equal(t, childErr, unwrapedErr)
	})

	t.Run("can wrap multiple errors and unwrap them", func(t *testing.T) {
		childErr1 := errforge.New("hello world")
		assert.Nil(t, errforge.UnwrapErrors(childErr1))
		childErr2 := errforge.New("hello univers")
		err := errforge.Wrap("error message", childErr1, childErr2)
		require.Error(t, err)
		assert.Equal(t, "error message", err.Error())

		unwrapedErrs := errforge.UnwrapErrors(err)
		assert.Equal(t, []error{childErr1, childErr2}, unwrapedErrs)
	})

	t.Run("should filter out nil errors from a list", func(t *testing.T) {
		childErr1 := errforge.New("hello world")
		childErr2 := errforge.New("hello univers")
		err := errforge.Wrap("error message", nil, childErr1, nil, childErr2, nil, nil)
		require.Error(t, err)
		assert.Equal(t, "error message", err.Error())

		unwrapedErrs := errforge.UnwrapErrors(err)
		assert.Equal(t, []error{childErr1, childErr2}, unwrapedErrs)
	})

	t.Run("should return nil when the only error is nil", func(t *testing.T) {
		assert.NoError(t, errforge.Wrap("error message", nil))
	})

	t.Run("should return nil when all the errors are nil", func(t *testing.T) {
		assert.NoError(t, errforge.Wrap("error message", nil, nil, nil))
	})

	t.Run("should return nil when you wrap no error", func(t *testing.T) {
		assert.NoError(t, errforge.Wrap("error message"))
	})
}

func TestWrappedErrorsWithFormat(t *testing.T) {
	t.Run("can be wrapped in another error and unwrapped", func(t *testing.T) {
		childErr := errforge.New("hello world")
		require.NoError(t, errforge.Unwrap(childErr))

		err := errforge.Wrapf("error %s", "message", childErr)
		require.Error(t, err)
		assert.Equal(t, "error message", err.Error())

		unwrapedErr := errforge.Unwrap(err)
		assert.Equal(t, childErr, unwrapedErr)
	})

	t.Run("can wrap multiple errors and unwrap them", func(t *testing.T) {
		childErr1 := errforge.New("hello world")
		assert.Nil(t, errforge.UnwrapErrors(childErr1))
		childErr2 := errforge.New("hello univers")
		err := errforge.Wrapf("error %s", "message", childErr1, childErr2)
		require.Error(t, err)
		assert.Equal(t, "error message", err.Error())

		unwrapedErrs := errforge.UnwrapErrors(err)
		assert.Equal(t, []error{childErr1, childErr2}, unwrapedErrs)
	})

	t.Run("should filter out nil errors from a list", func(t *testing.T) {
		childErr1 := errforge.New("hello world")
		childErr2 := errforge.New("hello univers")
		err := errforge.Wrapf("error %s", "message", nil, childErr1, nil, childErr2, nil, nil)
		require.Error(t, err)
		assert.Equal(t, "error message", err.Error())

		unwrapedErrs := errforge.UnwrapErrors(err)
		assert.Equal(t, []error{childErr1, childErr2}, unwrapedErrs)
	})

	t.Run("should return nil when the only error is nil", func(t *testing.T) {
		assert.NoError(t, errforge.Wrapf("error %s", "message", nil))
	})

	t.Run("should return nil when all the errors are nil", func(t *testing.T) {
		assert.NoError(t, errforge.Wrapf("error %s", "message", nil, nil, nil))
	})

	t.Run("should return nil when you wrap no error", func(t *testing.T) {
		assert.NoError(t, errforge.Wrapf("error %s", "message"))
	})

	t.Run("renders fmt's bad-verb marker when a format argument has the wrong type", func(t *testing.T) {
		childErr := errforge.New("hello world")
		err := errforge.Wrapf("hello %d", "not a number", childErr)
		require.Error(t, err)
		assert.Equal(t, "hello %!d(string=not a number)", err.Error())
		assert.Equal(t, childErr, errforge.Unwrap(err))
	})

	t.Run("ignores a trailing argument that is not an error", func(t *testing.T) {
		// "this should have been an error" is not an error, so it is dropped,
		// leaving nothing to wrap.
		assert.NoError(t, errforge.Wrapf("boom", "this should have been an error"))
	})

	t.Run("ignores non-error trailing arguments but keeps the real errors", func(t *testing.T) {
		cause := errforge.New("real cause")
		err := errforge.Wrapf("boom", "noise", cause, 123)
		assert.Equal(t, "boom", err.Error())
		assert.True(t, errforge.Is(err, cause))
	})

	t.Run("preserves errforge.Is and errforge.As through the wrap", func(t *testing.T) {
		sentinel := errforge.New("sentinel")
		typed := &mainError{}
		err := errforge.Wrapf("context %d", 42, sentinel, typed)
		assert.Equal(t, "context 42", err.Error())
		assert.True(t, errforge.Is(err, sentinel))
		var target *mainError
		assert.True(t, errforge.As(err, &target))
	})
}

func TestLinkedError(t *testing.T) {
	t.Run("should link errors", func(t *testing.T) {
		mainErr := errforge.New("main error")
		causeErr := errforge.New("cause error")
		err := errforge.Link(mainErr, causeErr)
		require.Error(t, err)

		assert.Equal(t, "main error: cause error", err.Error())

		unwrapedErr := errforge.Unwrap(err)
		assert.Equal(t, causeErr, unwrapedErr)
	})

	t.Run("should be the main error if the cause error is nil", func(t *testing.T) {
		mainErr := errforge.New("main error")
		err := errforge.Link(mainErr, nil)
		assert.Equal(t, mainErr, err)
	})

	t.Run("should be the cause error if the main error is nil", func(t *testing.T) {
		causeErr := errforge.New("cause error")
		err := errforge.Link(nil, causeErr)
		assert.Equal(t, causeErr, err)
	})

	t.Run("should be nil if both errors are nil", func(t *testing.T) {
		assert.NoError(t, errforge.Link(nil, nil))
	})

	t.Run("should provide the IS interface", func(t *testing.T) {
		mainErr := errforge.New("main error")
		causeErr := errforge.New("cause error")
		otherErr := errforge.New("another error")
		err := errforge.Link(mainErr, causeErr)
		assert.True(t, errforge.Is(err, mainErr))
		assert.True(t, errforge.Is(err, causeErr))
		assert.False(t, errforge.Is(err, otherErr))
	})

	t.Run("should provide the AS interface", func(t *testing.T) {
		mainErr := &mainError{}
		causeErr := &causeError{}
		err := errforge.Link(mainErr, causeErr)
		var tmp1 *mainError
		assert.True(t, errforge.As(err, &tmp1))
		var tmp2 *causeError
		assert.True(t, errforge.As(err, &tmp2))
		var tmp3 *anotherError
		assert.False(t, errforge.As(err, &tmp3))
	})
}

func TestLinkedErrors(t *testing.T) {
	t.Run("should link errors", func(t *testing.T) {
		mainErr := errforge.New("main error")
		causeErr1 := errforge.New("cause error 1")
		causeErr2 := errforge.New("cause error 2")
		err := errforge.Link(mainErr, causeErr1, causeErr2)
		require.Error(t, err)

		assert.Equal(t, "main error: [cause error 1,cause error 2]", err.Error())

		unwrapedErrs := errforge.UnwrapErrors(err)
		assert.Equal(t, []error{causeErr1, causeErr2}, unwrapedErrs)
	})

	t.Run("should be the main error if the cause errors are nil", func(t *testing.T) {
		mainErr := errforge.New("main error")
		err := errforge.Link(mainErr, nil, nil)
		assert.Equal(t, mainErr, err)
	})

	t.Run("should be the cause error if the main error is nil", func(t *testing.T) {
		causeErr1 := errforge.New("cause error 1")
		causeErr2 := errforge.New("cause error 2")
		err := errforge.Link(nil, causeErr1, causeErr2)

		unwrapedErrs := errforge.UnwrapErrors(err)
		assert.Equal(t, []error{causeErr1, causeErr2}, unwrapedErrs)
	})

	t.Run("should be nil if both errors are nil", func(t *testing.T) {
		assert.NoError(t, errforge.Link(nil, nil, nil))
	})

	t.Run("should provide the IS interface", func(t *testing.T) {
		mainErr := errforge.New("main error")
		causeErr1 := errforge.New("cause error 1")
		causeErr2 := errforge.New("cause error 2")
		otherErr := errforge.New("another error")
		err := errforge.Link(mainErr, causeErr1, causeErr2)
		assert.True(t, errforge.Is(err, mainErr))
		assert.True(t, errforge.Is(err, causeErr1))
		assert.True(t, errforge.Is(err, causeErr2))
		assert.False(t, errforge.Is(err, otherErr))
	})

	t.Run("should provide the AS interface", func(t *testing.T) {
		mainErr := &mainError{}
		causeErr1 := &causeError{}
		causeErr2 := &causeError{}
		err := errforge.Link(mainErr, causeErr1, causeErr2)
		var tmp1 *mainError
		assert.True(t, errforge.As(err, &tmp1))
		var tmp2 *causeError
		assert.True(t, errforge.As(err, &tmp2))
		var tmp3 *anotherError
		assert.False(t, errforge.As(err, &tmp3))
	})
}

// func TestWrap(t *testing.T) {
// 	tests := []struct {
// 		err     error
// 		wrapper error
// 		want    string
// 		wantNil bool
// 	}{
// 		{err: Message("err"), wrapper: Message("wrapper"), want: "wrapper: err"},
// 		{err: io.EOF, wrapper: Message("wrapper"), want: "wrapper: EOF"},
// 		{err: nil, wrapper: Message("wrapper"), wantNil: true},
// 		{err: Message("err"), wrapper: nil, want: "err"},
// 	}
// 	for n, tt := range tests {
// 		t.Run(fmt.Sprintf("case-%d", n+1), func(t *testing.T) {
// 			got := WithWrapper(tt.wrapper, tt.err)
// 			switch {
// 			case tt.wantNil:
// 				if got != nil {
// 					t.Errorf("WithWrapper(%#v, %#v): expected nil", tt.wrapper, tt.err)
// 				}
// 			default:
// 				if got.Error() != tt.want {
// 					t.Errorf("WithWrapper(%#v, %#v): got: %q, want %q", tt.wrapper, tt.err, got, tt.want)
// 				}
// 				if len(StackTrace(got)) != 0 {
// 					t.Errorf("WithWrapper(%#v, %#v): returned error must not contain a stack trace", tt.wrapper, tt.err)
// 				}
// 				if !errforge.Is(got, tt.err) {
// 					t.Errorf("WithWrapper(%#v, %#v): errforge.Is must return true for err", tt.wrapper, tt.err)
// 				}
// 				if tt.wrapper != nil && !errforge.Is(got, tt.wrapper) {
// 					t.Errorf("WithWrapper(%#v, %#v): errforge.Is must return true for wrapper", tt.wrapper, tt.err)
// 				}
// 				if tt.err != nil && !errforge.As(got, reflect.New(reflect.TypeOf(tt.err)).Interface()) {
// 					t.Errorf("errforge.As(WithWrapper(%#v, %#v), err): must return true for the err error type", tt.wrapper, tt.err)
// 				}
// 				if tt.wrapper != nil && !errforge.As(got, reflect.New(reflect.TypeOf(tt.wrapper)).Interface()) {
// 					t.Errorf("errforge.As(WithWrapper(%#v, %#v), err): must return true for the wrapper error type", tt.wrapper, tt.err)
// 				}
// 			}
// 		})
// 	}
// }

type mainError struct{}

func (*mainError) Error() string {
	return "main error"
}

type causeError struct{}

func (*causeError) Error() string {
	return "cause error"
}

type anotherError struct{}

func (*anotherError) Error() string {
	return "another error"
}
