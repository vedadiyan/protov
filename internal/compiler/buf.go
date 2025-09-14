package compiler

import (
	"github.com/bufbuild/protocompile/linker"
	"github.com/bufbuild/protocompile/parser"
	"github.com/bufbuild/protocompile/reporter"
	"go.lsp.dev/protocol"
)

type report struct {
	diagnostics         []protocol.Diagnostic
	syntaxMissing       map[string]bool
	pathToUnusedImports map[string]map[string]bool
}

// Error implements reporter.Handler for *diagnostics.
func (r *report) Error(err reporter.ErrorWithPos) error {
	r.diagnostics = append(r.diagnostics, newDiagnostic(err, false))
	return nil
}

// Warning implements reporter.Handler for *diagnostics.
func (r *report) Warning(err reporter.ErrorWithPos) {
	r.diagnostics = append(r.diagnostics, newDiagnostic(err, true))

	if err.Unwrap() == parser.ErrNoSyntax {
		if r.syntaxMissing == nil {
			r.syntaxMissing = make(map[string]bool)
		}
		r.syntaxMissing[err.GetPosition().Filename] = true
	} else if unusedImport, ok := err.Unwrap().(linker.ErrorUnusedImport); ok {
		if r.pathToUnusedImports == nil {
			r.pathToUnusedImports = make(map[string]map[string]bool)
		}

		path := err.GetPosition().Filename
		unused, ok := r.pathToUnusedImports[path]
		if !ok {
			unused = map[string]bool{}
			r.pathToUnusedImports[path] = unused
		}

		unused[unusedImport.UnusedImport()] = true
	}
}

func newDiagnostic(err reporter.ErrorWithPos, isWarning bool) protocol.Diagnostic {
	pos := protocol.Position{
		Line:      uint32(err.GetPosition().Line - 1),
		Character: uint32(err.GetPosition().Col - 1),
	}

	diagnostic := protocol.Diagnostic{
		// TODO: The compiler currently does not record spans for diagnostics. This is
		// essentially a bug that will result in worse diagnostics until fixed.
		Range:    protocol.Range{Start: pos, End: pos},
		Severity: protocol.DiagnosticSeverityError,
		Message:  err.Unwrap().Error(),
		Source:   "",
	}

	if isWarning {
		diagnostic.Severity = protocol.DiagnosticSeverityWarning
	}

	return diagnostic
}
