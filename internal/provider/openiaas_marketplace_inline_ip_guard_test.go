package provider

import (
	"go/ast"
	"testing"
)

// This guard parses the provider's own source and asserts, on the MARKETPLACE
// deploy branch of openIaasVirtualMachineCreate, two properties no behavioural
// unit test can reach without a live platform:
//
//  1. the NetworkDataMapping it builds sets IPAddress;
//  2. that branch runs the same two VPC preconditions as the template branch.
//
// Why (1) needs a guard. The client-side wire test
// (TestDeployOpenIaasItemNetworkData) pins the json tag, but it exercises the
// CLIENT. If the provider stops passing the configured address into the mapping,
// that test stays green, the deploy body goes out with no address, the platform
// assigns an arbitrary one and answers 201 — a deliberately chosen VPC static IP
// silently downgraded, with nothing wrong in the state and no diff to show it.
// ip_address is write-only, so no later refresh corrects it.
//
// Why (2) needs a guard. The address is only honoured on a VPC-backed network; on a
// plain network the platform discards it without error. And the platform assigns one
// explicit address per (virtual machine, network) pair, so two blocks pointing at
// the same network cannot both carry one — a refusal that, left to the platform,
// arrives from the deploy activity and leaves an untracked virtual machine behind.
// Both checks must therefore run BEFORE the deploy POST.
//
// History: this branch used to REFUSE ip_address outright, because
// NetworkDataMapping had no such field and the route had never been measured. The
// refusal was lifted once the marketplace module maintainer confirmed the deploy
// endpoint accepts ipAddress despite the published swagger omitting it. This guard
// is what keeps the replacement honest — do NOT weaken it; if it fails, restore the
// passthrough or the preconditions it names.
func TestOpenIaaSMarketplaceInlineIPIsWiredAndGuarded(t *testing.T) {
	const file = "resource_compute_iaas_opensource_virtual_machine.go"
	fn := findFuncDecl(t, file, "openIaasVirtualMachineCreate")

	branch := marketplaceBranchBody(fn)
	if branch == nil {
		t.Fatalf("could not find the marketplace deploy branch (an if/else-if testing %q) in openIaasVirtualMachineCreate — renamed? Keep this guard in step with the code rather than deleting it", "marketplace_item_id")
	}

	// (1) the mapping must carry the address
	lit := compositeLitNamed(branch, "client", "NetworkDataMapping")
	if lit == nil {
		t.Fatalf("no client.NetworkDataMapping literal found in the marketplace branch")
	}
	if !compositeLitHasKey(lit, "IPAddress") {
		t.Fatalf(
			"the marketplace branch builds client.NetworkDataMapping WITHOUT IPAddress.\n" +
				"The deploy would then carry no address, the platform would assign an arbitrary\n" +
				"one and answer 201, and the chosen VPC static IP would be silently lost —\n" +
				"ip_address is write-only, so no refresh ever corrects it.\n" +
				"Fix: pass IPAddress into the mapping. Do not relax this test.")
	}

	// (2) both VPC preconditions must run on this branch, before the deploy
	for _, want := range []string{
		"rejectInlineAdapterIPSharedNetwork",
		"validateInlineAdapterIPsTargetVPC",
	} {
		if !callsFuncNamed(branch, want) {
			t.Fatalf(
				"the marketplace branch does not call %s before deploying.\n"+
					"Without it a configured ip_address can be sent to a network that discards it,\n"+
					"or two blocks can claim one address on the same network — a refusal that arrives\n"+
					"from the deploy activity and leaves an untracked virtual machine behind.\n"+
					"Fix: run the same preconditions the template branch runs.", want)
		}
	}
}

// marketplaceBranchBody returns the body of the if/else-if branch whose condition
// mentions the "marketplace_item_id" attribute, i.e. the marketplace deploy path.
func marketplaceBranchBody(fn *ast.FuncDecl) *ast.BlockStmt {
	var found *ast.BlockStmt
	ast.Inspect(fn, func(n ast.Node) bool {
		if found != nil {
			return false
		}
		ifStmt, ok := n.(*ast.IfStmt)
		if !ok || ifStmt.Cond == nil {
			return true
		}
		mentions := false
		ast.Inspect(ifStmt.Cond, func(c ast.Node) bool {
			if lit, ok := c.(*ast.BasicLit); ok && lit.Value == `"marketplace_item_id"` {
				mentions = true
				return false
			}
			return true
		})
		if mentions {
			found = ifStmt.Body
			return false
		}
		return true
	})
	return found
}

// compositeLitNamed finds a composite literal of type <pkg>.<name> inside a block.
func compositeLitNamed(block *ast.BlockStmt, pkg, name string) *ast.CompositeLit {
	var found *ast.CompositeLit
	ast.Inspect(block, func(n ast.Node) bool {
		if found != nil {
			return false
		}
		lit, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		sel, ok := lit.Type.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil || sel.Sel.Name != name {
			return true
		}
		if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == pkg {
			found = lit
			return false
		}
		return true
	})
	return found
}

// compositeLitHasKey reports whether a keyed composite literal sets the given field.
func compositeLitHasKey(lit *ast.CompositeLit, key string) bool {
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		if ident, ok := kv.Key.(*ast.Ident); ok && ident.Name == key {
			return true
		}
	}
	return false
}

// callsFuncNamed reports whether the block contains a call to the named function.
func callsFuncNamed(block *ast.BlockStmt, name string) bool {
	found := false
	ast.Inspect(block, func(n ast.Node) bool {
		if found {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == name {
			found = true
			return false
		}
		return true
	})
	return found
}
