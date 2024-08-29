// The linter command runs the errforge analyzers: noerrors reports uses of
// the standard errors package and nofmterrorf reports calls to fmt.Errorf.
// Both suggest, and apply with -fix, the equivalent errforge call.
//
// Usage:
//
//	go run github.com/grandper/go-errforge/cmd/linter@latest ./...
//	go run github.com/grandper/go-errforge/cmd/linter@latest -fix ./...
//
// The binary also works as a go vet tool:
//
//	go vet -vettool=$(which linter) ./...
package main

import (
	"golang.org/x/tools/go/analysis/multichecker"

	"github.com/grandper/go-errforge/internal/rules"
)

func main() {
	multichecker.Main(rules.NoErrors, rules.NoFmtErrorf)
}
