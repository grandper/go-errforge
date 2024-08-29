package errforge_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grandper/go-errforge/errforge"
)

var (
	errRequired  = errforge.New("cannot be empty")
	errBadFormat = errforge.New("has a bad format")
)

// validateUser is the shape of a validation function that reports every
// failing input at once.
func validateUser(name, email string) error {
	var err error
	if name == "" {
		err = errforge.Append(err, errforge.FieldError("name", errRequired))
	}
	if email == "" || email[len(email)-1] == '@' {
		err = errforge.Append(err, errforge.FieldError("email", errBadFormat.WithDetail(email)))
	}
	return errforge.PropagateWithCode(err, errforge.InvalidArgument, "invalid user")
}

func TestFieldError(t *testing.T) {
	t.Run("marks an error with a field without changing its message", func(t *testing.T) {
		err := errforge.FieldError("name", errRequired)
		assert.Equal(t, "cannot be empty", err.Error())
	})

	t.Run("returns nil when the error is nil", func(t *testing.T) {
		assert.NoError(t, errforge.FieldError("name", nil))
	})

	t.Run("preserves identity for Is and Unwrap", func(t *testing.T) {
		err := errforge.FieldError("name", errRequired)
		assert.True(t, errforge.Is(err, errRequired))
		assert.Equal(t, errRequired, errforge.Unwrap(err))
	})

	t.Run("is transparent for Print", func(t *testing.T) {
		err := errforge.Wrap("invalid user", errforge.FieldError("name", errRequired))
		assert.Equal(t, "Error: invalid user\nCaused by: cannot be empty\n", errforge.Sprint(err))
	})
}

func TestGetFieldErrors(t *testing.T) {
	t.Run("collects every field in append order", func(t *testing.T) {
		err := validateUser("", "bob@")

		violations := errforge.GetFieldErrors(err)

		require.Len(t, violations, 2)
		assert.Equal(t, "name", violations[0].Field)
		assert.True(t, errforge.Is(violations[0].Err, errRequired))
		assert.Equal(t, "email", violations[1].Field)
		assert.True(t, errforge.Is(violations[1].Err, errBadFormat))
		assert.Equal(t, "has a bad format: bob@", violations[1].Err.Error())
	})

	t.Run("hands back the wrapped error, not the marker", func(t *testing.T) {
		violations := errforge.GetFieldErrors(errforge.FieldError("name", errRequired))
		require.Len(t, violations, 1)
		assert.Same(t, errRequired, violations[0].Err)
	})

	t.Run("joins nested fields with a dot", func(t *testing.T) {
		err := errforge.FieldError("address", errforge.FieldError("city", errRequired))

		violations := errforge.GetFieldErrors(err)

		require.Len(t, violations, 1)
		assert.Equal(t, "address.city", violations[0].Field)
		assert.Same(t, errRequired, violations[0].Err)
	})

	t.Run("reports a field wrapped inside a field separately when a message sits between them", func(t *testing.T) {
		err := errforge.FieldError("address", errforge.Wrap("bad address", errforge.FieldError("city", errRequired)))

		violations := errforge.GetFieldErrors(err)

		require.Len(t, violations, 2)
		assert.Equal(t, "address", violations[0].Field)
		assert.Equal(t, "bad address", violations[0].Err.Error())
		assert.Equal(t, "city", violations[1].Field)
	})

	t.Run("finds fields under wrappers and markers", func(t *testing.T) {
		err := errforge.WithStack(
			errforge.ToTransient(errforge.Wrap("outer", errforge.FieldError("name", errRequired))),
		)
		require.Len(t, errforge.GetFieldErrors(err), 1)
	})

	t.Run("returns nil when there is no field", func(t *testing.T) {
		assert.Nil(t, errforge.GetFieldErrors(errforge.New("boom")))
	})

	t.Run("returns nil for a nil error", func(t *testing.T) {
		assert.Nil(t, errforge.GetFieldErrors(nil))
	})
}

func TestFieldErrorKeepsItsCodeAndDetails(t *testing.T) {
	fieldPublic := &errforge.PublicError{Key: "validation.required", Message: "this field is required"}
	field := errforge.FieldError("name", errforge.Detailed("cannot be empty",
		errforge.WithCode(errforge.OutOfRange),
		errforge.WithPublic(fieldPublic),
	))

	t.Run("a field's PublicError is not the request's", func(t *testing.T) {
		err := errforge.PropagateWithCode(errforge.Append(nil, field), errforge.InvalidArgument, "invalid user")
		assert.Nil(t, errforge.GetDetails(err))
		assert.Equal(t, errforge.InvalidArgument, errforge.GetCode(err))
	})

	t.Run("a field's code does not surface through an uncoded Propagate", func(t *testing.T) {
		err := errforge.Propagate(errforge.Append(nil, field), "invalid user")
		assert.Equal(t, errforge.NoCode, errforge.GetCode(err))
	})

	t.Run("the field itself answers them once unwrapped", func(t *testing.T) {
		violations := errforge.GetFieldErrors(field)
		require.Len(t, violations, 1)
		assert.Same(t, fieldPublic, errforge.GetDetails(violations[0].Err))
		assert.Equal(t, errforge.OutOfRange, errforge.GetCode(violations[0].Err))
	})

	t.Run("the request's own PublicError still wins", func(t *testing.T) {
		requestPublic := &errforge.PublicError{Message: "please check the form"}
		err := errforge.Detailed("invalid user",
			errforge.WithCode(errforge.InvalidArgument),
			errforge.WithCause(errforge.Append(nil, field)),
			errforge.WithPublic(requestPublic),
		)
		assert.Same(t, requestPublic, errforge.GetDetails(err))
	})
}

func TestWriteErrorJSONFields(t *testing.T) {
	t.Run("lists every failing field with its message", func(t *testing.T) {
		rec := httptest.NewRecorder()

		errforge.WriteErrorJSON(rec, validateUser("", "bob@"))

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.JSONEq(t, `{
			"code":"INVALID_ARGUMENT",
			"message":"invalid user",
			"fields":[
				{"field":"name","message":"cannot be empty"},
				{"field":"email","message":"has a bad format: bob@"}
			]
		}`, rec.Body.String())
	})

	t.Run("renders a field's PublicError with its key and params", func(t *testing.T) {
		rec := httptest.NewRecorder()
		field := errforge.FieldError("quantity", errforge.Detailed("quantity exceeds stock",
			errforge.WithPublic(&errforge.PublicError{
				Key:     "validation.max",
				Message: "at most 5",
				Params:  errforge.Params{"max": 5},
			}),
		))
		err := errforge.PropagateWithCode(field, errforge.InvalidArgument, "invalid order")

		errforge.WriteErrorJSON(rec, err)

		assert.JSONEq(t, `{
			"code":"INVALID_ARGUMENT",
			"message":"invalid order",
			"fields":[{"field":"quantity","key":"validation.max","message":"at most 5","params":{"max":5}}]
		}`, rec.Body.String())
	})

	t.Run("omits fields when there are none", func(t *testing.T) {
		rec := httptest.NewRecorder()
		errforge.WriteErrorJSON(rec, errforge.NewWithCode(errforge.NotFound, "track 42 not found"))
		assert.JSONEq(t, `{"code":"NOT_FOUND","message":"track 42 not found"}`, rec.Body.String())
	})
}

func ExampleFieldError() {
	var (
		required  = errforge.New("cannot be empty")
		badFormat = errforge.New("has a bad format")
	)

	validate := func(name, email string) error {
		var err error
		if name == "" {
			err = errforge.Append(err, errforge.FieldError("name", required))
		}
		if email == "bob@" {
			err = errforge.Append(err, errforge.FieldError("email", badFormat.WithDetail(email)))
		}
		return errforge.PropagateWithCode(err, errforge.InvalidArgument, "invalid user")
	}

	err := validate("", "bob@")
	for _, v := range errforge.GetFieldErrors(err) {
		fmt.Println(v.Field, "-", v.Err, "- required:", errforge.Is(v.Err, required))
	}
	// Output:
	// name - cannot be empty - required: true
	// email - has a bad format: bob@ - required: false
}
