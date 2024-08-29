package rules_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/grandper/go-errforge/internal/rules"
)

func TestClassify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		format string
		nargs  int
		want   rules.Plan
	}{
		{"an error occurred", 0, rules.Plan{Kind: rules.LayoutNew, Text: "an error occurred"}},
		{"100%% sure", 0, rules.Plan{Kind: rules.LayoutNewf, Text: "100%% sure"}},
		{"reading %s", 1, rules.Plan{Kind: rules.LayoutNewf, Text: "reading %s"}},
		{"line %-4d of %q", 2, rules.Plan{Kind: rules.LayoutNewf, Text: "line %-4d of %q"}},
		{"%w", 1, rules.Plan{Kind: rules.LayoutIdentity}},
		{"opening: %w", 1, rules.Plan{Kind: rules.LayoutWrap, Text: "opening"}},
		{"reading %s: %w", 2, rules.Plan{Kind: rules.LayoutWrapf, Text: "reading %s"}},
		{"100%% sure: %w", 1, rules.Plan{Kind: rules.LayoutWrapf, Text: "100%% sure"}},
		{"invalid: %w, %w", 2, rules.Plan{Kind: rules.LayoutWrap, Text: "invalid"}},
		{"invalid: %w; %w", 2, rules.Plan{Kind: rules.LayoutWrap, Text: "invalid"}},
		{"invalid: %w %w", 2, rules.Plan{Kind: rules.LayoutWrap, Text: "invalid"}},
		{"%w: %w", 2, rules.Plan{Kind: rules.LayoutLead, Text: "%w"}},
		{"%w: user", 1, rules.Plan{Kind: rules.LayoutLead, Text: "user"}},
		{"%w: user %d", 2, rules.Plan{Kind: rules.LayoutLead, Text: "user %d"}},
		// No errforge function names these relationships.
		{"%w while reading %s", 2, rules.Plan{Kind: rules.LayoutUnsupported}},
		{"loading: %w: %w", 2, rules.Plan{Kind: rules.LayoutUnsupported}},
		{"%w: %w, %w", 3, rules.Plan{Kind: rules.LayoutUnsupported}},
		{"%w, %w", 2, rules.Plan{Kind: rules.LayoutUnsupported}},
		{"opening:%w", 1, rules.Plan{Kind: rules.LayoutUnsupported}},
		{"opening: %w!", 1, rules.Plan{Kind: rules.LayoutUnsupported}},
		{": %w", 1, rules.Plan{Kind: rules.LayoutUnsupported}},
		// Features the rewrite does not handle.
		{"%[1]s %[1]s", 1, rules.Plan{Kind: rules.LayoutUnsupported}},
		{"%*d", 2, rules.Plan{Kind: rules.LayoutUnsupported}},
		{"dangling %", 0, rules.Plan{Kind: rules.LayoutUnsupported}},
		// Argument count does not match the format: vet reports it, we do not rewrite it.
		{"reading %s", 0, rules.Plan{Kind: rules.LayoutUnsupported}},
		{"reading", 1, rules.Plan{Kind: rules.LayoutUnsupported}},
	}
	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, rules.Classify(tt.format, tt.nargs))
		})
	}
}
