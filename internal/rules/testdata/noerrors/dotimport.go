package noerrors

// A single-line dot import that the rewrite leaves unused: the rule deletes
// the declaration itself, since the driver only drops unused named imports.

import . "errors"

var ErrC = New("c") // want `do not use the errors package: use errforge\.New$`
