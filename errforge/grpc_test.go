package errforge_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/grandper/go-errforge/errforge"
)

func TestGRPCStatus(t *testing.T) {
	t.Run("maps the code to its gRPC code and uses the error message", func(t *testing.T) {
		err := errforge.GRPCStatus(errforge.NewWithCode(errforge.NotFound, "user 42 not found"))

		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
		assert.Equal(t, "user 42 not found", st.Message())
	})

	t.Run("uses the PublicError message when present", func(t *testing.T) {
		err := errforge.GRPCStatus(errforge.Detailed("internal failure",
			errforge.WithCode(errforge.InvalidArgument),
			errforge.WithPublic(&errforge.PublicError{Message: "bad request"}),
		))

		st, _ := status.FromError(err)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Equal(t, "bad request", st.Message())
	})

	t.Run("falls back to Unknown for an error without a code", func(t *testing.T) {
		err := errforge.GRPCStatus(errforge.New("boom"))

		st, _ := status.FromError(err)
		assert.Equal(t, codes.Unknown, st.Code())
	})

	t.Run("returns nil when the error is nil", func(t *testing.T) {
		assert.NoError(t, errforge.GRPCStatus(nil))
	})

	t.Run("surfaces a code set deep in the chain after propagation", func(t *testing.T) {
		deep := errforge.NewWithCode(errforge.ResourceExhausted, "throttled")
		err := errforge.GRPCStatus(errforge.Propagate(deep, "layer 1"))

		st, _ := status.FromError(err)
		assert.Equal(t, codes.ResourceExhausted, st.Code())
	})

	t.Run("forwards the code of an error received from a gRPC client", func(t *testing.T) {
		downstream := status.Error(codes.PermissionDenied, "not the owner")
		err := errforge.GRPCStatus(errforge.Propagate(downstream, "calling the catalog service"))

		st, _ := status.FromError(err)
		assert.Equal(t, codes.PermissionDenied, st.Code())
		assert.Equal(t, "calling the catalog service", st.Message())
	})

	t.Run("answers a context error with its own gRPC code", func(t *testing.T) {
		st, _ := status.FromError(errforge.GRPCStatus(context.DeadlineExceeded))
		assert.Equal(t, codes.DeadlineExceeded, st.Code())

		st, _ = status.FromError(errforge.GRPCStatus(errforge.Wrap("searching", context.Canceled)))
		assert.Equal(t, codes.Canceled, st.Code())
	})
}
