package rules_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/grandper/go-errforge/internal/rules"
)

func TestNoFmtErrorf(t *testing.T) {
	analysistest.Run(t, moduleRoot(t), rules.NoFmtErrorf, noFmtErrorfTestdata)
}

func TestNoFmtErrorfSuggestedFixes(t *testing.T) {
	analysistest.RunWithSuggestedFixes(t, moduleRoot(t), rules.NoFmtErrorf, noFmtErrorfTestdata)
}
