package rules_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/grandper/go-errforge/internal/rules"
)

func TestNoErrors(t *testing.T) {
	analysistest.Run(t, moduleRoot(t), rules.NoErrors, noErrorsTestdata)
}

func TestNoErrorsSuggestedFixes(t *testing.T) {
	analysistest.RunWithSuggestedFixes(t, moduleRoot(t), rules.NoErrors, noErrorsTestdata)
}
