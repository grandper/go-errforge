package errforge_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/grandper/go-errforge/errforge"
)

func TestSprint(t *testing.T) {
	t.Run("returns empty string for nil", func(t *testing.T) {
		assert.Empty(t, errforge.Sprint(nil))
	})

	t.Run("renders a single error", func(t *testing.T) {
		assert.Equal(t, "Error: boom\n", errforge.Sprint(errforge.New("boom")))
	})

	t.Run("renders a wrapped chain with Caused by", func(t *testing.T) {
		inner := errforge.New("connection refused")
		outer := errforge.Wrap("query failed", inner)
		want := "Error: query failed\nCaused by: connection refused\n"
		assert.Equal(t, want, errforge.Sprint(outer))
	})

	t.Run("enumerates multierror members", func(t *testing.T) {
		err := errforge.Join(errforge.New("a"), errforge.New("b"))
		want := "Error: errors occurred: [a, b]\n\t1. Error: a\n\t2. Error: b\n"
		assert.Equal(t, want, errforge.Sprint(err))
	})

	t.Run("does not print a redundant WithStack wrapper twice", func(t *testing.T) {
		err := errforge.WithStack(errforge.New("boom"))
		out := errforge.Sprint(err)
		// The message appears once, followed by the stack; no "Caused by".
		assert.Equal(t, 1, strings.Count(out, "boom"))
		assert.NotContains(t, out, "Caused by")
	})

	t.Run("includes the stack trace of the error", func(t *testing.T) {
		err := errforge.WithStack(errforge.New("boom"))
		out := errforge.Sprint(err)
		assert.True(t, strings.HasPrefix(out, "Error: boom\n"), out)
		assert.Contains(t, out, "\tat errforge_test.TestSprint", out)
	})

	t.Run("does not repeat a line for a marker over a Propagate", func(t *testing.T) {
		cause := errforge.New("dial tcp: refused")
		err := errforge.ToTransient(errforge.Propagate(cause, "calling payment service"))
		want := "Error: calling payment service\nCaused by: dial tcp: refused\n"
		assert.Equal(t, want, errforge.Sprint(err))
	})

	t.Run("collapses a run of markers into one entry with its stack", func(t *testing.T) {
		cause := errforge.New("dial tcp: refused")
		err := errforge.ToTransient(errforge.WithStack(errforge.Propagate(cause, "calling payment service")))
		out := errforge.Sprint(err)
		assert.Equal(t, 1, strings.Count(out, "calling payment service"), out)
		assert.Equal(t, 1, strings.Count(out, "\tat errforge_test.TestSprint"), out)
		assert.True(t, strings.HasSuffix(out, "Caused by: dial tcp: refused\n"), out)
	})

	t.Run("prints a stack trace only under the node that carries it", func(t *testing.T) {
		err := errforge.Wrap("query failed", errforge.WithStack(errforge.New("boom")))
		out := errforge.Sprint(err)
		assert.Equal(t, 1, strings.Count(out, "\tat errforge_test.TestSprint"), out)
		assert.True(t, strings.HasPrefix(out, "Error: query failed\nCaused by: boom\n\tat "), out)
	})

	t.Run("keeps every message above a nested multierror", func(t *testing.T) {
		err := errforge.Wrap("ctx", errforge.Wrap("db", errforge.Join(errforge.New("a"), errforge.New("b"))))
		want := "Error: ctx\nCaused by: db\nCaused by: errors occurred: [a, b]\n\t1. Error: a\n\t2. Error: b\n"
		assert.Equal(t, want, errforge.Sprint(err))
	})
}

func TestPrint(t *testing.T) {
	// Print writes to stderr; make sure it does not panic for nil or a chain.
	assert.NotPanics(t, func() {
		errforge.Print(nil)
		errforge.Print(errforge.Wrap("ctx", errforge.WithStack(errforge.New("boom"))))
	})
}
