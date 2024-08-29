package errforge_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grandper/go-errforge/errforge"
)

var (
	errEmptyName          = errors.New("empty name")
	errMaxLengthExceeded  = errors.New("max length exceeded")
	errTrackNotFound      = errors.New("track not found")
	errUnregisteredOutage = errors.New("unregistered outage")
)

func TestMapper(t *testing.T) {
	t.Run("maps a registered error to its code", func(t *testing.T) {
		mapper := errforge.NewMapper(errforge.Map(errforge.NotFound).To(errTrackNotFound))

		assert.Equal(t, errforge.NotFound, mapper.Code(errTrackNotFound))
	})

	t.Run("maps every error registered to the same code", func(t *testing.T) {
		mapper := errforge.NewMapper(
			errforge.Map(errforge.FailedPrecondition).To(errEmptyName, errMaxLengthExceeded),
		)

		assert.Equal(t, errforge.FailedPrecondition, mapper.Code(errEmptyName))
		assert.Equal(t, errforge.FailedPrecondition, mapper.Code(errMaxLengthExceeded))
	})

	t.Run("maps errors registered to different codes", func(t *testing.T) {
		mapper := errforge.NewMapper(
			errforge.Map(errforge.FailedPrecondition).To(errEmptyName),
			errforge.Map(errforge.NotFound).To(errTrackNotFound),
		)

		assert.Equal(t, errforge.FailedPrecondition, mapper.Code(errEmptyName))
		assert.Equal(t, errforge.NotFound, mapper.Code(errTrackNotFound))
	})

	t.Run("maps a wrapped error to the code of the error it wraps", func(t *testing.T) {
		mapper := errforge.NewMapper(errforge.Map(errforge.NotFound).To(errTrackNotFound))

		wrapped := fmt.Errorf("failed to load track: %w", errTrackNotFound)

		assert.Equal(t, errforge.NotFound, mapper.Code(wrapped))
	})

	t.Run("maps an unregistered error to internal", func(t *testing.T) {
		mapper := errforge.NewMapper(errforge.Map(errforge.NotFound).To(errTrackNotFound))

		assert.Equal(t, errforge.Internal, mapper.Code(errUnregisteredOutage))
	})

	t.Run("maps a nil error to OK", func(t *testing.T) {
		mapper := errforge.NewMapper()

		assert.Equal(t, errforge.OK, mapper.Code(nil))
	})

	t.Run("uses the last registration when an error is registered several times", func(t *testing.T) {
		mapper := errforge.NewMapper(
			errforge.Map(errforge.NotFound).To(errTrackNotFound),
			errforge.Map(errforge.FailedPrecondition).To(errTrackNotFound),
		)

		assert.Equal(t, errforge.FailedPrecondition, mapper.Code(errTrackNotFound))
	})

	t.Run("ignores nil errors on registration", func(t *testing.T) {
		mapper := errforge.NewMapper(errforge.Map(errforge.NotFound).To(nil))

		assert.Equal(t, errforge.OK, mapper.Code(nil))
		assert.Equal(t, errforge.Internal, mapper.Code(errUnregisteredOutage))
	})

	t.Run("maps context errors by default", func(t *testing.T) {
		mapper := errforge.NewMapper()

		assert.Equal(t, errforge.Canceled, mapper.Code(context.Canceled))
		assert.Equal(t, errforge.DeadlineExceeded, mapper.Code(context.DeadlineExceeded))
	})

	t.Run("lets an option override a default context mapping", func(t *testing.T) {
		mapper := errforge.NewMapper(errforge.Map(errforge.Unavailable).To(context.DeadlineExceeded))

		assert.Equal(t, errforge.Unavailable, mapper.Code(context.DeadlineExceeded))
	})
}

func TestMapperAttachCodeTo(t *testing.T) {
	mapper := errforge.NewMapper(errforge.Map(errforge.NotFound).To(errTrackNotFound))

	t.Run("attaches the mapped code", func(t *testing.T) {
		err := mapper.AttachCodeTo(errTrackNotFound)

		assert.Equal(t, errforge.NotFound, errforge.GetCode(err))
	})

	t.Run("attaches the code through a wrapper", func(t *testing.T) {
		err := mapper.AttachCodeTo(fmt.Errorf("failed to load track: %w", errTrackNotFound))

		assert.Equal(t, errforge.NotFound, errforge.GetCode(err))
	})

	t.Run("attaches internal to an unregistered error", func(t *testing.T) {
		err := mapper.AttachCodeTo(errUnregisteredOutage)

		assert.Equal(t, errforge.Internal, errforge.GetCode(err))
	})

	t.Run("preserves the message and the identity", func(t *testing.T) {
		err := mapper.AttachCodeTo(errTrackNotFound)

		assert.Equal(t, errTrackNotFound.Error(), err.Error())
		require.ErrorIs(t, err, errTrackNotFound)
		assert.Same(t, errTrackNotFound, errors.Unwrap(err))
	})

	t.Run("returns nil for a nil error", func(t *testing.T) {
		assert.NoError(t, mapper.AttachCodeTo(nil))
	})

	t.Run("keeps a code already in the chain", func(t *testing.T) {
		coded := errforge.PropagateWithCode(errTrackNotFound, errforge.AlreadyExists, "duplicate")

		err := mapper.AttachCodeTo(coded)

		assert.Same(t, coded, err)
		assert.Equal(t, errforge.AlreadyExists, errforge.GetCode(err))
	})

	t.Run("keeps a context error as is", func(t *testing.T) {
		err := mapper.AttachCodeTo(context.DeadlineExceeded)

		assert.Equal(t, context.DeadlineExceeded, err)
		assert.Equal(t, errforge.DeadlineExceeded, errforge.GetCode(err))
	})

	t.Run("reaches the transport boundary", func(t *testing.T) {
		rec := httptest.NewRecorder()
		errforge.WriteErrorJSON(rec, mapper.AttachCodeTo(errTrackNotFound))

		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.JSONEq(t, `{"code":"NOT_FOUND","message":"track not found"}`, rec.Body.String())
	})

	t.Run("marks a transient code as transient", func(t *testing.T) {
		transientMapper := errforge.NewMapper(errforge.Map(errforge.Unavailable).To(errUnregisteredOutage))

		assert.True(t, errforge.IsTransient(transientMapper.AttachCodeTo(errUnregisteredOutage)))
	})
}
