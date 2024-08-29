package errforge_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/grandper/go-errforge/errforge"
)

// fields returns the names GetFieldErrors reports for err, in order. It is
// the one getter that gathers a fact from every node instead of stopping at
// the first match, so tagging the nodes of a tree with FieldError exposes the
// order the package walks it: the node itself, then its cause, then its
// members, each in turn.
func fields(err error) []string {
	var out []string
	for _, v := range errforge.GetFieldErrors(err) {
		out = append(out, v.Field)
	}
	return out
}

func TestWalkOrder(t *testing.T) {
	t.Run("visits nothing for a nil error", func(t *testing.T) {
		assert.Empty(t, fields(nil))
	})

	t.Run("visits a lone node once", func(t *testing.T) {
		assert.Equal(t, []string{"boom"}, fields(errforge.FieldError("boom", errforge.New("boom"))))
	})

	t.Run("visits the node, then its cause, then the members in order", func(t *testing.T) {
		err := errforge.FieldError("top", errforge.Wrap("top",
			errforge.FieldError("join", errforge.Join(
				errforge.FieldError("a", errforge.New("a")),
				errforge.FieldError("b", errforge.Wrap("b",
					errforge.FieldError("c", errforge.New("c")),
				)),
			)),
		))
		assert.Equal(t, []string{"top", "join", "a", "b", "c"}, fields(err))
	})

	t.Run("steps through the transparent markers instead of skipping them", func(t *testing.T) {
		err := errforge.FieldError("outer", errforge.ToTransient(errforge.WithStack(
			errforge.FieldError("inner", errforge.New("boom")),
		)))
		assert.Equal(t, []string{"outer", "inner"}, fields(err))
	})

	t.Run("follows a Link into the joined error before its cause", func(t *testing.T) {
		err := errforge.Link(
			errforge.FieldError("left", errforge.Wrap("left", errforge.FieldError("l", errforge.New("l")))),
			errforge.FieldError("right", errforge.New("right")),
		)
		assert.Equal(t, []string{"left", "l", "right"}, fields(err))
	})

	t.Run("follows a multi-cause Link into the joined error before its causes", func(t *testing.T) {
		err := errforge.Link(
			errforge.FieldError("left", errforge.New("left")),
			errforge.FieldError("r1", errforge.New("r1")),
			errforge.FieldError("r2", errforge.New("r2")),
		)
		assert.Equal(t, []string{"left", "r1", "r2"}, fields(err))
	})
}
