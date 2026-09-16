package provider

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// idSettersFromCreateActivity are the two helpers that resolve the resource id
// from a create activity. Each of their call sites in computeVirtualMachineCreate
// marks one of the four asynchronous create modes (clone, content library,
// marketplace, from scratch).
var idSettersFromCreateActivity = map[string]bool{
	"setIdFromActivityState":          true,
	"setIdFromActivityConcernedItems": true,
}

// expectedCreateModes is the number of asynchronous create modes the resource
// exposes. Pinning it makes the guard fail loudly if a mode is added (and must
// be wired) or removed (and the constant must follow) — never vacuously pass.
const expectedCreateModes = 4

// TestVMCreateFailurePathsRouteThroughRecovery is the wiring guard for the
// anti-orphan recovery.
//
// The behavioural tests exercise recoverVMCreateFailure directly, so they would
// stay GREEN if someone reverted a create branch to a bare
// `return diag.Errorf(...)` — and that revert is precisely the defect: it leaves
// a virtual machine that exists platform-side absent from the Terraform state,
// unreachable by `terraform destroy`.
//
// This guard closes that gap structurally. In computeVirtualMachineCreate, every
// call that resolves the id from a create activity must be immediately followed
// by an `if err != nil` block returning through recoverVMCreateFailure. An AST
// walk (not a string scan) is used so reformatting or line breaks cannot let a
// reverted branch slip through.
func TestVMCreateFailurePathsRouteThroughRecovery(t *testing.T) {
	const file = "resource_compute_virtual_machine.go"

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, 0)
	if err != nil {
		t.Fatalf("parsing %s: %v", file, err)
	}

	var createFn *ast.FuncDecl
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Name.Name == "computeVirtualMachineCreate" {
			createFn = fn
			break
		}
	}
	if createFn == nil || createFn.Body == nil {
		t.Fatalf("computeVirtualMachineCreate not found in %s — the guard would vacuously pass", file)
	}

	var wired, offenders []string
	ast.Inspect(createFn.Body, func(n ast.Node) bool {
		block, ok := n.(*ast.BlockStmt)
		if !ok {
			return true
		}
		for i, stmt := range block.List {
			setter, ok := idSetterCall(stmt)
			if !ok {
				continue
			}
			where := fmt.Sprintf("%s at %s", setter, fset.Position(stmt.Pos()))
			if i+1 < len(block.List) && returnsThroughRecovery(block.List[i+1]) {
				wired = append(wired, where)
			} else {
				offenders = append(offenders, where)
			}
		}
		return true
	})

	if len(offenders) > 0 {
		t.Fatalf("a create branch resolves the id from its activity but does NOT route its failure through recoverVMCreateFailure — "+
			"a virtual machine created platform-side would be orphaned outside the Terraform state:\n  %v", offenders)
	}
	if len(wired) != expectedCreateModes {
		t.Fatalf("expected %d asynchronous create modes wired to recoverVMCreateFailure, found %d: %v",
			expectedCreateModes, len(wired), wired)
	}
}

// recoverVMCreateFailureFixedArgs is the number of non-variadic parameters of
// recoverVMCreateFailure (ctx, d, read, activity, activityID, name, action,
// cause). A call carrying more than that passes at least one exclusion id.
const recoverVMCreateFailureFixedArgs = 8

// TestCloneRecoveryExcludesItsSource guards the single most destructive
// regression this recovery could suffer.
//
// On the clone path the source virtual machine is a virtual machine too. If the
// platform lists it as a concerned item and the call site stops excluding it,
// the recovery adopts the SOURCE id — which SDKv2 persists as tainted, so the
// next `terraform apply` DESTROYS a virtual machine this resource never
// created. The behavioural tests cover the exclusion logic; only this guard
// covers the call site actually passing it.
func TestCloneRecoveryExcludesItsSource(t *testing.T) {
	const file = "resource_compute_virtual_machine.go"

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, 0)
	if err != nil {
		t.Fatalf("parsing %s: %v", file, err)
	}

	var checked int
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		fn, ok := call.Fun.(*ast.Ident)
		if !ok || fn.Name != "recoverVMCreateFailure" {
			return true
		}
		if !callActionMentionsClone(call) {
			return true
		}
		checked++
		if len(call.Args) <= recoverVMCreateFailureFixedArgs {
			t.Errorf("the clone recovery at %s passes no exclusion id: the clone SOURCE could be adopted, "+
				"tainted, and destroyed by the next apply — pass cloneVirtualMachineId", fset.Position(call.Pos()))
		}
		return true
	})

	if checked == 0 {
		t.Fatalf("no clone recovery call site found in %s — the guard would vacuously pass", file)
	}
}

// callActionMentionsClone reports whether one of the call's string literal
// arguments is the human-readable action of a clone operation.
func callActionMentionsClone(call *ast.CallExpr) bool {
	for _, arg := range call.Args {
		lit, ok := arg.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			continue
		}
		if strings.Contains(lit.Value, "clone") {
			return true
		}
	}
	return false
}

// idSetterCall reports whether stmt is a call to one of the id-resolving
// helpers, and which one.
func idSetterCall(stmt ast.Stmt) (string, bool) {
	expr, ok := stmt.(*ast.ExprStmt)
	if !ok {
		return "", false
	}
	call, ok := expr.X.(*ast.CallExpr)
	if !ok {
		return "", false
	}
	id, ok := call.Fun.(*ast.Ident)
	if !ok || !idSettersFromCreateActivity[id.Name] {
		return "", false
	}
	return id.Name, true
}

// returnsThroughRecovery reports whether stmt is an `if err != nil` block whose
// body returns a call to recoverVMCreateFailure.
func returnsThroughRecovery(stmt ast.Stmt) bool {
	ifStmt, ok := stmt.(*ast.IfStmt)
	if !ok || !isErrNotNil(ifStmt.Cond) || ifStmt.Body == nil {
		return false
	}
	for _, inner := range ifStmt.Body.List {
		ret, ok := inner.(*ast.ReturnStmt)
		if !ok || len(ret.Results) != 1 {
			continue
		}
		call, ok := ret.Results[0].(*ast.CallExpr)
		if !ok {
			continue
		}
		if fn, ok := call.Fun.(*ast.Ident); ok && fn.Name == "recoverVMCreateFailure" {
			return true
		}
	}
	return false
}

// isErrNotNil reports whether cond is the `err != nil` comparison.
func isErrNotNil(cond ast.Expr) bool {
	bin, ok := cond.(*ast.BinaryExpr)
	if !ok || bin.Op != token.NEQ {
		return false
	}
	lhs, ok := bin.X.(*ast.Ident)
	if !ok || lhs.Name != "err" {
		return false
	}
	rhs, ok := bin.Y.(*ast.Ident)
	return ok && rhs.Name == "nil"
}
