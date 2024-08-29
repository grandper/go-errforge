package rules

import (
	"bytes"
	"go/ast"
	"go/constant"
	"go/printer"
	"go/types"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/types/typeutil"
)

// NoFmtErrorf is a go/analysis analyzer that reports calls to fmt.Errorf and
// rewrites them to the errforge call naming the same relationship:
//
//	fmt.Errorf("message")                 -> errforge.New("message")
//	fmt.Errorf("reading %s", name)        -> errforge.Newf("reading %s", name)
//	fmt.Errorf("opening: %w", err)        -> errforge.Wrap("opening", err)
//	fmt.Errorf("reading %s: %w", n, err)  -> errforge.Wrapf("reading %s", n, err)
//	fmt.Errorf("invalid: %w, %w", a, b)   -> errforge.Wrap("invalid", a, b)
//	fmt.Errorf("%w: %w", ErrX, cause)     -> ErrX.WithErr(cause)          (ErrX is a *errforge.Error)
//	fmt.Errorf("%w: id %d", ErrX, id)     -> ErrX.WithDetailf("id %d", id)
//	fmt.Errorf("%w: %w", a, b)            -> errforge.Link(a, b)
//	fmt.Errorf("%w: id %d", io.EOF, id)   -> errforge.Link(io.EOF, errforge.Newf("id %d", id))
//	fmt.Errorf("%w", err)                 -> err
//
// A %w anywhere else, a non-constant format, or an argument count that does
// not match the format is reported without a fix.
//
//nolint:gochecknoglobals // go/analysis requires the analyzer to be a package-level var
var NoFmtErrorf = &analysis.Analyzer{
	Name: noFmtErrorfName,
	Doc:  "reports calls to fmt.Errorf and rewrites them to the equivalent errforge call",
	Run:  runNoFmtErrorf,
}

const (
	noFmtErrorfName   = "nofmterrorf"
	noFmtErrorfPrefix = "do not use fmt.Errorf: "
	noFmtErrorfByHand = noFmtErrorfPrefix +
		"no direct errforge equivalent for this format, choose Wrap, Link, Propagate or WithErr by hand"
)

func runNoFmtErrorf(pass *analysis.Pass) (any, error) {
	if pass.Pkg.Path() == errforgePath {
		// errforge is the one place that legitimately wraps fmt.Errorf.
		//nolint:nilnil // go/analysis analyzers return (nil, nil) when producing no result
		return nil, nil
	}
	for _, file := range pass.Files {
		ref := resolveErrforge(file)
		var diags []analysis.Diagnostic
		ast.Inspect(file, func(node ast.Node) bool {
			if call, ok := node.(*ast.CallExpr); ok && isFmtErrorf(pass, call) {
				diags = append(diags, fmtErrorfDiagnostic(pass, call, ref))
			}
			return true
		})
		dropDotImport(pass, file, "fmt", func(member string) bool { return member == "Errorf" }, diags)
		for _, diag := range diags {
			pass.Report(diag)
		}
	}
	//nolint:nilnil // go/analysis analyzers return (nil, nil) when producing no result
	return nil, nil
}

func isFmtErrorf(pass *analysis.Pass, call *ast.CallExpr) bool {
	fn, ok := typeutil.Callee(pass.TypesInfo, call).(*types.Func)
	return ok && fn.Pkg() != nil && fn.Pkg().Path() == "fmt" && fn.Name() == "Errorf"
}

func fmtErrorfDiagnostic(pass *analysis.Pass, call *ast.CallExpr, ref errforgeRef) analysis.Diagnostic {
	diag := analysis.Diagnostic{Pos: call.Pos(), End: call.End(), Category: noFmtErrorfName}
	format, ok := constantFormat(pass, call)
	if !ok || call.Ellipsis.IsValid() {
		diag.Message = noFmtErrorfPrefix + "the format is not a constant, choose the errforge function by hand"
		return diag
	}
	fix, ok := rewriteFmtErrorf(pass, call, ref, Classify(format, len(call.Args)-1))
	if !ok {
		diag.Message = noFmtErrorfByHand
		return diag
	}
	diag.Message = noFmtErrorfPrefix + "use " + fix.target
	if fix.usesErrforge {
		fix.edits = append(fix.edits, ref.edits...)
	}
	diag.SuggestedFixes = []analysis.SuggestedFix{{Message: "Replace with " + fix.target, TextEdits: fix.edits}}
	return diag
}

// constantFormat returns the value of the call's format argument when it is
// a constant string: a literal or a named constant.
func constantFormat(pass *analysis.Pass, call *ast.CallExpr) (string, bool) {
	if len(call.Args) == 0 {
		return "", false
	}
	tv, ok := pass.TypesInfo.Types[call.Args[0]]
	if !ok || tv.Value == nil || tv.Value.Kind() != constant.String {
		return "", false
	}
	return constant.StringVal(tv.Value), true
}

// fix is the rewrite of one fmt.Errorf call.
type fix struct {
	target       string // what the diagnostic tells the author to use
	edits        []analysis.TextEdit
	usesErrforge bool // whether the rewritten code refers to the errforge package
}

func rewriteFmtErrorf(pass *analysis.Pass, call *ast.CallExpr, ref errforgeRef, p Plan) (fix, bool) {
	newFormat := replace(call.Args[0], strconv.Quote(p.Text))
	switch p.Kind {
	case LayoutNew:
		return errforgeCall(call, ref, "New"), true
	case LayoutNewf:
		return errforgeCall(call, ref, "Newf"), true
	case LayoutIdentity:
		return fix{
			target: "the wrapped error itself",
			edits:  []analysis.TextEdit{replace(call, exprText(pass, call.Args[1]))},
		}, true
	case LayoutWrap:
		f := errforgeCall(call, ref, "Wrap")
		f.edits = append(f.edits, newFormat)
		return f, true
	case LayoutWrapf:
		f := errforgeCall(call, ref, "Wrapf")
		f.edits = append(f.edits, newFormat)
		return f, true
	case LayoutLead:
		return rewriteLeading(pass, call, ref, p.Text), true
	case LayoutUnsupported:
		return fix{}, false
	default:
		return fix{}, false
	}
}

// errforgeCall keeps the arguments in place and only renames the function.
func errforgeCall(call *ast.CallExpr, ref errforgeRef, name string) fix {
	target := ref.member(name)
	return fix{target: target, edits: []analysis.TextEdit{replace(call.Fun, target)}, usesErrforge: true}
}

// rewriteLeading handles "%w: rest": the first argument is a sentinel or a
// cause rendered in front of the message. A *errforge.Error gets its own
// builders; anything else is linked to the message.
func rewriteLeading(pass *analysis.Pass, call *ast.CallExpr, ref errforgeRef, rest string) fix {
	head := exprText(pass, call.Args[1])
	rest2 := call.Args[2:] // the arguments of the part after "%w: "
	restArgs := make([]string, 0, len(rest2))
	for _, arg := range rest2 {
		restArgs = append(restArgs, exprText(pass, arg))
	}
	sentinel := isErrforgeError(pass.TypesInfo.TypeOf(call.Args[1]))
	formatted := strings.Contains(rest, "%")

	var target, text string
	switch {
	case rest == "%w" && sentinel:
		target = head + ".WithErr"
		text = target + "(" + restArgs[0] + ")"
	case rest == "%w":
		target = ref.member("Link")
		text = target + "(" + head + ", " + restArgs[0] + ")"
	case sentinel && !formatted:
		target = head + ".WithDetail"
		text = target + "(" + strconv.Quote(rest) + ")"
	case sentinel:
		target = head + ".WithDetailf"
		text = target + "(" + strings.Join(append([]string{strconv.Quote(rest)}, restArgs...), ", ") + ")"
	case !formatted:
		target = ref.member("Link")
		text = target + "(" + head + ", " + ref.member("New") + "(" + strconv.Quote(rest) + "))"
	default:
		target = ref.member("Link")
		inner := ref.member("Newf") + "(" + strings.Join(append([]string{strconv.Quote(rest)}, restArgs...), ", ") + ")"
		text = target + "(" + head + ", " + inner + ")"
	}
	return fix{target: target, edits: []analysis.TextEdit{replace(call, text)}, usesErrforge: !sentinel}
}

// isErrforgeError reports whether t is *errforge.Error, the type of a
// sentinel created with errforge.New.
func isErrforgeError(t types.Type) bool {
	ptr, ok := t.(*types.Pointer)
	if !ok {
		return false
	}
	named, ok := ptr.Elem().(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj.Pkg() != nil && obj.Pkg().Path() == errforgePath && obj.Name() == "Error"
}

// exprText renders an expression as source code.
func exprText(pass *analysis.Pass, expr ast.Expr) string {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, pass.Fset, expr); err != nil {
		return ""
	}
	return buf.String()
}
