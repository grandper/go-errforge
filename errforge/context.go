package errforge

import (
	"context"
)

// FromContext returns the error associated with ctx if it has been canceled or
// has exceeded its deadline (context.Canceled or context.DeadlineExceeded).
// It returns nil if the context is still active.
func FromContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
