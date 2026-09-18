package provider

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// This guard parses the provider's own source with go/ast and asserts a structural
// invariant that no behavioural test can cover: that every create/update entry point
// able to carry an inline VPC static IP installs the deferred rollback, with a NAMED
// diagnostics return, BEFORE its first return statement.
//
// Why the invariant needs a guard. `ip_address` is write-only, so the read path
// preserves whatever the state holds; and terraform-plugin-sdk/v2 persists the
// PLANNED values when a create or update returns an error. One error return that does
// not roll the addresses back is enough to record an address that was never applied —
// after which state equals config, Terraform sees no diff, the function is never
// called again, and the address is never applied. Permanent, and invisible in a plan.
//
// Why a DEFER and not a wrapper per return. Wrapping each return was tried:
// adversarial review found five separate forgotten spots, each narrower than the
// last, and text-scanning for unwrapped returns could still miss `diag.FromErr`, an
// indirect diagnostics variable, or a new return elsewhere in the call tree. The
// deferred guard acts on what the function RETURNS, so it covers every present and
// future error path. This test is what keeps it installed.
//
// Why AST rather than string matching: a `defer` inside a comment or a string literal
// must not satisfy the check, and the defer must come before any return to be of any
// use at all.
//
// If this test fails, do NOT weaken it: add
// `defer rollbackInlineAdapterIPsOnAnyError(d, &diags)` as the first statement of the
// named-return function it names.
func TestInlineAdapterIPRollbackIsInstalledOnEveryEntryPoint(t *testing.T) {
	const rollbackFunc = "rollbackInlineAdapterIPsOnAnyError"

	cases := []struct {
		file     string
		function string
		why      string
	}{
		{
			file:     "resource_compute_iaas_opensource_virtual_machine.go",
			function: "openIaasVirtualMachineUpdate",
			why:      "it reconciles the inline VPC static IP and has many error returns before and after it",
		},
		{
			file:     "resource_compute_iaas_opensource_virtual_machine.go",
			function: "openIaasVirtualMachineCreate",
			why:      "it sends ipAddress in the create body and can fail after the VM exists",
		},
		{
			file:     "resource_compute_virtual_machine.go",
			function: "updateVirtualMachine",
			why:      "it applies the address through the adapter patch and has many error returns around it",
		},
		{
			file:     "resource_compute_virtual_machine.go",
			function: "computeVirtualMachineCreate",
			why:      "it can fail after the VM is deployed but before the address is applied",
		},
	}

	for _, tc := range cases {
		t.Run(tc.function, func(t *testing.T) {
			fn := findFuncDecl(t, tc.file, tc.function)

			// (1) The deferred guard can only change what the function returns if the
			// result is NAMED, and the defer must take the address of THAT name — a
			// defer given some other variable would compile and do nothing useful.
			resultName := namedDiagnosticsResultName(fn)
			if resultName == "" {
				t.Fatalf("%s does not declare a NAMED diag.Diagnostics result, so a deferred rollback could not change what it returns", tc.function)
			}

			// (2) The defer must be installed, and BEFORE any return — a defer placed
			// after an early return would leave that return uncovered.
			deferAt, firstReturnAt := -1, -1
			for i, stmt := range fn.Body.List {
				if deferAt < 0 && isDeferCallTo(stmt, rollbackFunc, resultName) {
					deferAt = i
				}
				if firstReturnAt < 0 && containsReturn(stmt) {
					firstReturnAt = i
				}
			}
			if deferAt < 0 {
				t.Fatalf(
					"%s in %s does not install the deferred inline-ip rollback.\n"+
						"Reason it must: %s. ip_address is write-only, so the SDK persists the planned\n"+
						"value on an error, the read preserves it, state == config, and no future diff\n"+
						"ever brings this function back — the address is never applied.\n"+
						"Fix: add `defer %s(d, &diags)` at the top of the function. Do not relax this test.",
					tc.function, tc.file, tc.why, rollbackFunc,
				)
			}
			if firstReturnAt >= 0 && firstReturnAt < deferAt {
				t.Fatalf(
					"%s installs the rollback defer at statement %d, AFTER a return at statement %d: that early return is not covered. Move the defer to the top of the function.",
					tc.function, deferAt, firstReturnAt,
				)
			}
		})
	}
}

// findFuncDecl parses one source file and returns the named top-level function.
func findFuncDecl(t *testing.T, file, name string) *ast.FuncDecl {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parsing %s: %v", file, err)
	}
	for _, decl := range parsed.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil || fn.Name.Name != name || fn.Body == nil {
			continue
		}
		return fn
	}
	t.Fatalf("function %s not found in %s — renamed? Keep this guard in step with the code rather than deleting it", name, file)
	return nil
}

// namedDiagnosticsResultName returns the NAME of the function's named
// diag.Diagnostics result, or "" when it has none. The name matters: the defer has to
// take the address of that exact identifier to be able to change what is returned.
func namedDiagnosticsResultName(fn *ast.FuncDecl) string {
	if fn.Type.Results == nil {
		return ""
	}
	for _, field := range fn.Type.Results.List {
		if len(field.Names) == 0 {
			continue // an unnamed result cannot be mutated by a defer
		}
		sel, ok := field.Type.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil || sel.Sel.Name != "Diagnostics" {
			continue
		}
		pkg, ok := sel.X.(*ast.Ident)
		if !ok || pkg.Name != "diag" {
			continue
		}
		return field.Names[0].Name
	}
	return ""
}

// isDeferCallTo reports whether a statement is exactly
// `defer <name>(d, &<resultName>)`. Both arguments are checked: a defer handed some
// other variable would compile and quietly do nothing for the returned diagnostics.
func isDeferCallTo(stmt ast.Stmt, name, resultName string) bool {
	deferStmt, ok := stmt.(*ast.DeferStmt)
	if !ok || deferStmt.Call == nil {
		return false
	}
	fun, ok := deferStmt.Call.Fun.(*ast.Ident)
	if !ok || fun.Name != name {
		return false
	}
	if len(deferStmt.Call.Args) != 2 {
		return false
	}
	first, ok := deferStmt.Call.Args[0].(*ast.Ident)
	if !ok || first.Name != "d" {
		return false
	}
	unary, ok := deferStmt.Call.Args[1].(*ast.UnaryExpr)
	if !ok || unary.Op != token.AND {
		return false
	}
	arg, ok := unary.X.(*ast.Ident)
	return ok && arg.Name == resultName
}

// containsReturn reports whether a statement is, or contains, a return of the
// ENCLOSING function. A return inside a nested function literal belongs to that
// literal, not to the guarded function, so it is not an early exit.
func containsReturn(stmt ast.Stmt) bool {
	found := false
	ast.Inspect(stmt, func(n ast.Node) bool {
		if found {
			return false
		}
		if _, isFuncLit := n.(*ast.FuncLit); isFuncLit {
			return false
		}
		if _, isReturn := n.(*ast.ReturnStmt); isReturn {
			found = true
			return false
		}
		return true
	})
	return found
}
