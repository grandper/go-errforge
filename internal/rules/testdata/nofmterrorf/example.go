// Package nofmterrorf exercises every layout the nofmterrorf rule rewrites.
// The file already imports errforge and keeps using fmt for Sprintf, so the
// fix must neither add nor remove an import.
package nofmterrorf

import (
	"fmt"
	"io"

	"github.com/grandper/go-errforge/errforge"
)

var ErrNotFound = errforge.New("not found")

const message = "constant message"

func plain() error {
	return fmt.Errorf("an error occurred") // want `do not use fmt.Errorf: use errforge\.New$`
}

func constant() error {
	return fmt.Errorf(message) // want `do not use fmt.Errorf: use errforge\.New$`
}

func formatted(name string) error {
	return fmt.Errorf("reading %s", name) // want `do not use fmt.Errorf: use errforge\.Newf$`
}

func percent() error {
	return fmt.Errorf("100%% sure") // want `do not use fmt.Errorf: use errforge\.Newf$`
}

func wrap(err error) error {
	return fmt.Errorf("opening resource: %w", err) // want `do not use fmt.Errorf: use errforge\.Wrap$`
}

func wrapf(name string, err error) error {
	return fmt.Errorf("reading %s: %w", name, err) // want `do not use fmt.Errorf: use errforge\.Wrapf$`
}

func wrapMany(a, b error) error {
	return fmt.Errorf("invalid user: %w, %w", a, b) // want `do not use fmt.Errorf: use errforge\.Wrap$`
}

func link(a, b error) error {
	return fmt.Errorf("%w: %w", a, b) // want `do not use fmt.Errorf: use errforge\.Link$`
}

func withErr(err error) error {
	return fmt.Errorf("%w: %w", ErrNotFound, err) // want `do not use fmt.Errorf: use ErrNotFound\.WithErr$`
}

func withDetail() error {
	return fmt.Errorf("%w: user", ErrNotFound) // want `do not use fmt.Errorf: use ErrNotFound\.WithDetail$`
}

func withDetailf(id int) error {
	return fmt.Errorf("%w: user %d", ErrNotFound, id) // want `do not use fmt.Errorf: use ErrNotFound\.WithDetailf$`
}

func linkNew() error {
	return fmt.Errorf("%w: unexpected", io.EOF) // want `do not use fmt.Errorf: use errforge\.Link$`
}

func linkNewf(id int) error {
	return fmt.Errorf("%w: user %d", io.EOF, id) // want `do not use fmt.Errorf: use errforge\.Link$`
}

func identity(err error) error {
	return fmt.Errorf("%w", err) // want `do not use fmt.Errorf: use the wrapped error itself$`
}

func middle(err error, name string) error {
	return fmt.Errorf("%w while reading %s", err, name) // want `no direct errforge equivalent for this format`
}

func chained(a, b error) error {
	return fmt.Errorf("loading: %w: %w", a, b) // want `no direct errforge equivalent for this format`
}

func indexed(name string) error {
	return fmt.Errorf("%[1]s %[1]s", name) // want `no direct errforge equivalent for this format`
}

func dynamic(format string) error {
	return fmt.Errorf(format) // want `the format is not a constant`
}

func spread(args []any) error {
	return fmt.Errorf("a %s and a %s", args...) // want `the format is not a constant`
}

func keepsFmt() string {
	return fmt.Sprintf("%d", 1)
}
