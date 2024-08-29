package nofmterrorf

// The file uses fmt only through an alias and only for Errorf, and does not
// import errforge: the fix adds the errforge import once for the whole file,
// and the driver drops the fmt import the rewrite leaves unused.

import (
	format "fmt"
)

func aliased() error {
	return format.Errorf("an error occurred") // want `do not use fmt.Errorf: use errforge\.New$`
}

func aliasedWrap(err error) error {
	return format.Errorf("closing: %w", err) // want `do not use fmt.Errorf: use errforge\.Wrap$`
}
