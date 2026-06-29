package client

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// TestComputeReadsUseNotFound404 is the exhaustive guard for the #384 PR-B flip:
// every requireNotFoundOrOK call in the compute client surface (compute_*.go,
// excluding tests) must pass 404 as the not-found code, NEVER 403. It walks the
// AST of each compute_*.go source file and fails if any call still passes the
// literal 403 — so reverting ANY of the 39 flipped sites is caught here,
// including the list/special sites that the behavioral TestComputeNotFound404Flip
// does not individually exercise.
//
// An AST walk (not a string scan) is used so reformatting, line breaks or
// whitespace cannot make a reverted 403 slip through.
//
// Conscious exceptions (NOT compute reads under the 403->404 flip):
//   - GuestOperatingSystem.Read passes 500 (a separate 500-as-absent pathology,
//     tracked as its own follow-up); 500 != 403, so it is not flagged.
//   - object_storage_*.go reads remain at 403 (the 500-on-absent API bug); they
//     are not compute_*.go files, so they are out of this scan by construction.
func TestComputeReadsUseNotFound404(t *testing.T) {
	matches, err := filepath.Glob("compute_*.go")
	if err != nil {
		t.Fatalf("globbing compute_*.go: %v", err)
	}
	var files []string
	for _, f := range matches {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		files = append(files, f)
	}
	if len(files) == 0 {
		t.Fatal("no compute_*.go source files found — the guard would vacuously pass; check the test working directory")
	}

	fset := token.NewFileSet()
	var offenders []string
	for _, file := range files {
		f, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			t.Fatalf("parsing %s: %v", file, err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			id, ok := call.Fun.(*ast.Ident)
			if !ok || id.Name != "requireNotFoundOrOK" || len(call.Args) != 2 {
				return true
			}
			lit, ok := call.Args[1].(*ast.BasicLit)
			if !ok {
				return true // a non-literal not-found code is not the 403 we guard against
			}
			if lit.Value == "403" {
				offenders = append(offenders, fset.Position(call.Pos()).String())
			}
			return true
		})
	}
	if len(offenders) > 0 {
		t.Fatalf("since #384 every compute requireNotFoundOrOK call must pass 404 (not 403); found %d site(s) still passing 403: %v", len(offenders), offenders)
	}
}
