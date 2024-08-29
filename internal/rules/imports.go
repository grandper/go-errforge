package rules

import (
	"go/ast"
	"go/token"
	"strconv"

	"golang.org/x/tools/go/analysis"
)

const (
	// errforgePath is the import path of the package the rules rewrite to.
	errforgePath = "github.com/grandper/go-errforge/errforge"
	// errforgeName is the name the package is imported under by default.
	errforgeName = "errforge"
)

// errforgeRef describes how a file refers to the errforge package: the
// qualifier to prefix its members with, and the edits that add the import
// when the file does not have it yet.
type errforgeRef struct {
	qualifier string // "errforge." normally, "" under a dot import
	edits     []analysis.TextEdit
}

// member returns the expression naming the errforge member in the file.
func (r errforgeRef) member(name string) string {
	return r.qualifier + name
}

// resolveErrforge finds how file imports errforge, or returns the edit
// adding the import when it does not.
func resolveErrforge(file *ast.File) errforgeRef {
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil || path != errforgePath {
			continue
		}
		switch {
		case spec.Name == nil:
			return errforgeRef{qualifier: errforgeName + "."}
		case spec.Name.Name == ".":
			return errforgeRef{}
		case spec.Name.Name == "_":
			continue
		default:
			return errforgeRef{qualifier: spec.Name.Name + "."}
		}
	}
	return errforgeRef{qualifier: errforgeName + ".", edits: []analysis.TextEdit{addImport(file)}}
}

// addImport returns the edit inserting the errforge import. The position
// depends only on the file, so every diagnostic of every rule produces the
// same edit and the driver merges them into a single import. Imports the
// rewrite leaves unused (fmt, errors) are removed by the driver when it
// formats the fixed file.
func addImport(file *ast.File) analysis.TextEdit {
	quoted := strconv.Quote(errforgePath)
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.IMPORT {
			continue
		}
		if gen.Lparen.IsValid() {
			return insertAt(gen.Lparen+1, "\n\t"+quoted)
		}
		return insertAt(gen.End(), "\nimport "+quoted)
	}
	return insertAt(file.Name.End(), "\n\nimport "+quoted)
}

// dropDotImport removes the dot import of pkgPath from file when the fixes
// leave it unused: every member the file uses is accepted by fixable, and
// every reported use got a fix. The driver removes the named imports a fix
// leaves unused, but not a dot import, whose members it cannot tell from the
// file's own names. The edit is added to every fix; identical edits merge.
func dropDotImport(
	pass *analysis.Pass, file *ast.File, pkgPath string, fixable func(member string) bool, diags []analysis.Diagnostic,
) {
	edit, ok := dotImportEdit(pass, file, pkgPath)
	if !ok || usesOtherMember(pass, file, pkgPath, fixable) {
		return
	}
	for i := range diags {
		if len(diags[i].SuggestedFixes) == 0 {
			return // a use stays, and so must the import
		}
	}
	for i := range diags {
		fix := &diags[i].SuggestedFixes[0]
		fix.TextEdits = append(fix.TextEdits, edit)
	}
}

// dotImportEdit returns the edit deleting the dot import of pkgPath, if any.
func dotImportEdit(pass *analysis.Pass, file *ast.File, pkgPath string) (analysis.TextEdit, bool) {
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.IMPORT {
			continue
		}
		for _, spec := range gen.Specs {
			imp, isImport := spec.(*ast.ImportSpec)
			if !isImport || imp.Name == nil || imp.Name.Name != "." {
				continue
			}
			if path, err := strconv.Unquote(imp.Path.Value); err != nil || path != pkgPath {
				continue
			}
			if !gen.Lparen.IsValid() {
				// A single-line declaration; the errforge import is inserted right after it.
				return analysis.TextEdit{Pos: gen.Pos(), End: gen.End()}, true
			}
			return deleteLine(pass.Fset, imp), true
		}
	}
	return analysis.TextEdit{}, false
}

// usesOtherMember reports whether file uses a member of the package that
// fixable does not accept.
func usesOtherMember(pass *analysis.Pass, file *ast.File, pkgPath string, fixable func(member string) bool) bool {
	tf := pass.Fset.File(file.Pos())
	for ident, obj := range pass.TypesInfo.Uses {
		if pass.Fset.File(ident.Pos()) != tf || obj.Pkg() == nil || obj.Pkg().Path() != pkgPath {
			continue
		}
		if !fixable(obj.Name()) {
			return true
		}
	}
	return false
}

// deleteLine returns the edit removing the whole line holding node.
func deleteLine(fset *token.FileSet, node ast.Node) analysis.TextEdit {
	tf := fset.File(node.Pos())
	line := tf.Line(node.Pos())
	end := node.End()
	if line < tf.LineCount() {
		end = tf.LineStart(line + 1)
	}
	return analysis.TextEdit{Pos: tf.LineStart(line), End: end}
}

func insertAt(pos token.Pos, text string) analysis.TextEdit {
	return analysis.TextEdit{Pos: pos, End: pos, NewText: []byte(text)}
}

func replace(node ast.Node, text string) analysis.TextEdit {
	return analysis.TextEdit{Pos: node.Pos(), End: node.End(), NewText: []byte(text)}
}
