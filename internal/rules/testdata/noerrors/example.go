// Package noerrors exercises the noerrors rule. The file does not import
// errforge, so the fix adds the import once; errors stays imported because
// ErrUnsupported has no equivalent and is left in place.
package noerrors

import (
	"errors"
	"fmt"
)

var ErrA = errors.New("a") // want `do not use the errors package: use errforge\.New$`

type myError struct{}

func (*myError) Error() string { return "my error" }

func is(err error) bool {
	return errors.Is(err, ErrA) // want `do not use the errors package: use errforge\.Is$`
}

func as(err error) bool {
	var target *myError
	return errors.As(err, &target) // want `do not use the errors package: use errforge\.As$`
}

func unwrap(err error) error {
	return errors.Unwrap(err) // want `do not use the errors package: use errforge\.Unwrap$`
}

func join(a, b error) error {
	return errors.Join(a, b) // want `do not use the errors package: use errforge\.Join$`
}

func unsupported() error {
	return errors.ErrUnsupported // want `do not use the errors package: errors\.ErrUnsupported has no errforge equivalent`
}

func unrelated() {
	fmt.Println("fmt is not the concern of this rule")
}
