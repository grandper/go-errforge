package rules

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// NoErrors is a go/analysis analyzer that reports every use of the standard
// errors package. The members errforge exposes under the same name — New,
// Is, As, Unwrap and Join — are rewritten to the errforge call; the others
// (errors.ErrUnsupported) are reported without a fix.
//
//nolint:gochecknoglobals // go/analysis requires the analyzer to be a package-level var
var NoErrors = &analysis.Analyzer{
	Name: noErrorsName,
	Doc:  "reports uses of the standard errors package and rewrites them to the equivalent errforge call",
	Run:  runNoErrors,
}

const (
	noErrorsName   = "noerrors"
	noErrorsPrefix = "do not use the errors package: "
)

// errorsEquivalent reports whether errforge exposes the errors member under
// the same name, with the same signature.
func errorsEquivalent(member string) bool {
	switch member {
	case "New", "Is", "As", "Unwrap", "Join":
		return true
	default:
		return false
	}
}

func runNoErrors(pass *analysis.Pass) (any, error) {
	if pass.Pkg.Path() == errforgePath {
		// errforge is the one place that legitimately wraps the errors package.
		//nolint:nilnil // go/analysis analyzers return (nil, nil) when producing no result
		return nil, nil
	}
	for _, file := range pass.Files {
		ref := resolveErrforge(file)
		var diags []analysis.Diagnostic
		ast.Inspect(file, func(node ast.Node) bool {
			switch node := node.(type) {
			case *ast.SelectorExpr:
				// errors.New, or stderrors.New under an alias.
				if isErrorsPackage(pass, node.X) {
					diags = append(diags, errorsDiagnostic(node, node.Sel.Name, ref))
					return false
				}
			case *ast.Ident:
				// New under a dot import.
				if obj := pass.TypesInfo.Uses[node]; obj != nil && isErrorsMember(obj) {
					diags = append(diags, errorsDiagnostic(node, node.Name, ref))
				}
			}
			return true
		})
		dropDotImport(pass, file, "errors", errorsEquivalent, diags)
		for _, diag := range diags {
			pass.Report(diag)
		}
	}
	//nolint:nilnil // go/analysis analyzers return (nil, nil) when producing no result
	return nil, nil
}

func isErrorsPackage(pass *analysis.Pass, expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return false
	}
	name, ok := pass.TypesInfo.Uses[ident].(*types.PkgName)
	return ok && name.Imported().Path() == "errors"
}

func isErrorsMember(obj types.Object) bool {
	return obj.Pkg() != nil && obj.Pkg().Path() == "errors"
}

func errorsDiagnostic(node ast.Node, member string, ref errforgeRef) analysis.Diagnostic {
	diag := analysis.Diagnostic{Pos: node.Pos(), End: node.End(), Category: noErrorsName}
	if !errorsEquivalent(member) {
		diag.Message = noErrorsPrefix + "errors." + member + " has no errforge equivalent"
		return diag
	}
	target := ref.member(member)
	diag.Message = noErrorsPrefix + "use " + target
	edits := append([]analysis.TextEdit{replace(node, target)}, ref.edits...)
	diag.SuggestedFixes = []analysis.SuggestedFix{{Message: "Replace with " + target, TextEdits: edits}}
	return diag
}
