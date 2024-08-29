package rules_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// moduleRoot returns the root of the repository, the directory analysistest
// loads the testdata packages from. Handing it the module root rather than
// the testdata directory makes it load the packages in module mode, as part
// of this module, so the testdata type-checks against the real errforge
// package. The Go tooling still ignores the testdata directory in ./...
// patterns; only the explicit patterns below reach it.
func moduleRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	return root
}

const (
	noErrorsTestdata    = "./internal/rules/testdata/noerrors"
	noFmtErrorfTestdata = "./internal/rules/testdata/nofmterrorf"
)
