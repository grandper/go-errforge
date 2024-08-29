package rules

import (
	"strings"
	"unicode/utf8"
)

// verb is one conversion of a format string.
type verb struct {
	letter     rune
	start, end int // byte offsets of the whole "%..." sequence in the format
}

// parseVerbs scans a format string and returns its conversions in order. It
// reports false when the format uses a feature the rewrite does not handle:
// an explicit argument index, a '*' width or precision, or a dangling '%'.
func parseVerbs(format string) ([]verb, bool) {
	var verbs []verb
	for i := 0; i < len(format); {
		if format[i] != '%' {
			i++
			continue
		}
		start := i
		i++ // the '%'
		if i < len(format) && format[i] == '%' {
			i++ // a literal percent
			continue
		}
		for i < len(format) && strings.ContainsRune("+-# 0123456789.", rune(format[i])) {
			i++ // flags, width and precision
		}
		if i >= len(format) || format[i] == '*' || format[i] == '[' {
			return nil, false
		}
		r, size := utf8.DecodeRuneInString(format[i:])
		i += size
		verbs = append(verbs, verb{letter: r, start: start, end: i})
	}
	return verbs, true
}

// Layout is the shape of a fmt.Errorf call, which decides the errforge call
// replacing it.
type Layout int

const (
	// LayoutUnsupported is a format whose %w verbs sit where no errforge
	// function names the relationship; the fix is left to the author.
	LayoutUnsupported Layout = iota
	// LayoutNew is a format with no verb and no percent:
	// "message" -> New("message").
	LayoutNew
	// LayoutNewf is a format with no %w:
	// "reading %s" -> Newf("reading %s", name).
	LayoutNewf
	// LayoutIdentity is "%w" alone: the call is the wrapped error itself.
	LayoutIdentity
	// LayoutWrap is "prefix: %w[, %w]*" with a verb-free prefix:
	// -> Wrap("prefix", errs...).
	LayoutWrap
	// LayoutWrapf is the same with a formatted prefix:
	// -> Wrapf("reading %s", name, errs...).
	LayoutWrapf
	// LayoutLead is "%w: rest", a sentinel or cause in front:
	// -> WithErr / WithDetail / WithDetailf on a *errforge.Error, Link otherwise.
	LayoutLead
)

// Plan is the rewrite chosen for a fmt.Errorf call.
type Plan struct {
	Kind Layout
	// Text is the new format: the prefix for Wrap and Wrapf, the part after
	// "%w: " for the leading layout.
	Text string
}

// causeSep separates a message from the cause it renders, as in "msg: %w".
const causeSep = ": "

// Classify decides how a fmt.Errorf call with the given constant format and
// number of arguments after the format is rewritten.
func Classify(format string, nargs int) Plan {
	verbs, ok := parseVerbs(format)
	if !ok || len(verbs) != nargs {
		return Plan{Kind: LayoutUnsupported}
	}
	wraps := 0
	for _, v := range verbs {
		if v.letter == 'w' {
			wraps++
		}
	}
	switch {
	case wraps == 0 && nargs == 0 && !strings.Contains(format, "%"):
		return Plan{Kind: LayoutNew, Text: format}
	case wraps == 0:
		return Plan{Kind: LayoutNewf, Text: format}
	case format == "%w":
		return Plan{Kind: LayoutIdentity}
	}
	if p, matched := trailing(format, verbs); matched {
		return p
	}
	if p, matched := leading(format, verbs); matched {
		return p
	}
	return Plan{Kind: LayoutUnsupported}
}

// trailing matches "prefix: %w", "prefix %s: %w" and "prefix: %w, %w": every
// %w comes last, after the message and separated from it by causeSep.
func trailing(format string, verbs []verb) (Plan, bool) {
	first := -1
	for i, v := range verbs {
		if v.letter == 'w' {
			first = i
			break
		}
	}
	for i := first; i < len(verbs); i++ {
		if verbs[i].letter != 'w' {
			return Plan{}, false
		}
		if i > first && !isSiblingSep(format[verbs[i-1].end:verbs[i].start]) {
			return Plan{}, false
		}
	}
	if verbs[len(verbs)-1].end != len(format) {
		return Plan{}, false
	}
	cut := verbs[first].start - len(causeSep)
	if cut <= 0 || format[cut:verbs[first].start] != causeSep {
		return Plan{}, false
	}
	prefix := format[:cut]
	if first == 0 && !strings.Contains(prefix, "%") {
		return Plan{Kind: LayoutWrap, Text: prefix}, true
	}
	return Plan{Kind: LayoutWrapf, Text: prefix}, true
}

// isSiblingSep reports whether sep separates two causes rendered side by
// side, as in "%w, %w".
func isSiblingSep(sep string) bool {
	switch sep {
	case ", ", "; ", " ":
		return true
	default:
		return false
	}
}

// leading matches "%w: rest": a sentinel or a cause in front of the message.
// rest holds no other %w, except when it is exactly "%w" ("%w: %w").
func leading(format string, verbs []verb) (Plan, bool) {
	head := "%w" + causeSep
	if !strings.HasPrefix(format, head) || len(format) == len(head) {
		return Plan{}, false
	}
	rest := format[len(head):]
	for _, v := range verbs[1:] {
		if v.letter == 'w' && rest != "%w" {
			return Plan{}, false
		}
	}
	return Plan{Kind: LayoutLead, Text: rest}, true
}
