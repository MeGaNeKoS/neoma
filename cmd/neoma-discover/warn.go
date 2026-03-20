package main

import (
	"fmt"
	"go/ast"
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

type remapWarning struct {
	HandlerName string
	CalleeName  string
	ToStatuses  []int
}

func (w remapWarning) String() string {
	var targets []string
	for _, s := range w.ToStatuses {
		if text := http.StatusText(s); text != "" {
			targets = append(targets, fmt.Sprintf("%d %s", s, text))
		} else {
			targets = append(targets, strconv.Itoa(s))
		}
	}
	return fmt.Sprintf("neoma-discover: warning: errors from %s being remapped to %s in %s",
		w.CalleeName, strings.Join(targets, ", "), w.HandlerName)
}

func detectRemappedErrors(
	handlerName string,
	pkg *packages.Package,
	body *ast.BlockStmt,
	allPkgs map[string]*packages.Package,
	errType *errorTypeInfo,
	statusField string,
) []remapWarning {
	if body == nil {
		return nil
	}

	var warnings []remapWarning

	for _, stmt := range body.List {
		assign, ok := stmt.(*ast.AssignStmt)
		if !ok || len(assign.Rhs) != 1 {
			continue
		}

		call, ok := assign.Rhs[0].(*ast.CallExpr)
		if !ok {
			continue
		}

		calleeName, calleePkg := resolveCallTarget(pkg, call)
		if calleeName == "" {
			continue
		}

		calleeErrors := discoverCalleeErrors(calleePkg, calleeName, allPkgs, errType, statusField)
		if len(calleeErrors) == 0 {
			continue
		}

		ifStmt := findIfErrNil(body, assign)
		if ifStmt == nil {
			continue
		}

		handlerErrors := discoverErrorsInBlock(pkg, ifStmt.Body, errType, statusField)
		if len(handlerErrors) == 0 {
			continue
		}

		calleeStatuses := map[int]bool{}
		for _, ce := range calleeErrors {
			calleeStatuses[ce.Status] = true
		}

		var remapped []int
		seen := map[int]bool{}
		for _, he := range handlerErrors {
			if !calleeStatuses[he.Status] && !seen[he.Status] {
				remapped = append(remapped, he.Status)
				seen[he.Status] = true
			}
		}

		if len(remapped) > 0 {
			warnings = append(warnings, remapWarning{
				HandlerName: handlerName,
				CalleeName:  calleeName,
				ToStatuses:  remapped,
			})
		}
	}

	return warnings
}

func discoverCalleeErrors(pkgPath, funcName string, allPkgs map[string]*packages.Package, errType *errorTypeInfo, statusField string) []discoveredError {
	visited := map[string]bool{}
	return followCall(pkgPath, funcName, allPkgs, visited, func(p *packages.Package, b *ast.BlockStmt, v map[string]bool) []discoveredError {
		return discoverErrors(p, b, allPkgs, v, errType, statusField)
	})
}

func discoverErrorsInBlock(pkg *packages.Package, block *ast.BlockStmt, errType *errorTypeInfo, statusField string) []discoveredError {
	var errs []discoveredError
	ast.Inspect(block, func(n ast.Node) bool {
		ret, ok := n.(*ast.ReturnStmt)
		if !ok {
			return true
		}
		for _, expr := range ret.Results {
			switch e := expr.(type) {
			case *ast.CallExpr:
				if callReturnsErrorType(pkg, e, errType) {
					status := extractStatusFromFirstArg(pkg, e)
					if status > 0 {
						errs = append(errs, discoveredError{Status: status})
					}
				}
			case *ast.UnaryExpr:
				if comp, ok := e.X.(*ast.CompositeLit); ok {
					info := compositeTypeInfo(pkg, comp)
					if info != nil && info.PkgPath == errType.PkgPath && info.TypeName == errType.TypeName {
						status := extractStatusFromComposite(pkg, comp, statusField)
						if status > 0 {
							errs = append(errs, discoveredError{Status: status})
						}
					}
				}
			}
		}
		return true
	})
	return errs
}

func findIfErrNil(block *ast.BlockStmt, after ast.Stmt) *ast.IfStmt {
	found := false
	for _, stmt := range block.List {
		if stmt == after {
			found = true
			continue
		}
		if !found {
			continue
		}
		ifStmt, ok := stmt.(*ast.IfStmt)
		if !ok {
			continue
		}
		if isErrNilCheck(ifStmt.Cond) {
			return ifStmt
		}
		break
	}
	return nil
}

func isErrNilCheck(expr ast.Expr) bool {
	bin, ok := expr.(*ast.BinaryExpr)
	if !ok {
		return false
	}
	if bin.Op.String() != "!=" {
		return false
	}
	ident, ok := bin.X.(*ast.Ident)
	if !ok {
		return false
	}
	nilIdent, ok := bin.Y.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == "err" && nilIdent.Name == "nil"
}
