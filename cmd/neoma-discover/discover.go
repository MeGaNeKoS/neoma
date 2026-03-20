package main

import (
	"go/ast"
	"go/token"
	"go/types"
	"net/http"

	"golang.org/x/tools/go/packages"
)

func discoverErrors(pkg *packages.Package, body *ast.BlockStmt, allPkgs map[string]*packages.Package, visited map[string]bool, errType *errorTypeInfo, statusField string) []discoveredError {
	if body == nil {
		return nil
	}

	var errs []discoveredError

	ast.Inspect(body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CompositeLit:
			info := compositeTypeInfo(pkg, node)
			if info != nil && info.PkgPath == errType.PkgPath && info.TypeName == errType.TypeName {
				status := extractStatusFromComposite(pkg, node, statusField)
				if status > 0 {
					errs = append(errs, discoveredError{
						Status: status,
						Title:  http.StatusText(status),
					})
				}
			}

		case *ast.CallExpr:
			funcName, pkgPath := resolveCallTarget(pkg, node)
			if funcName == "" {
				return true
			}

			if callReturnsErrorType(pkg, node, errType) || returnsErrorType(pkgPath, funcName, allPkgs, errType) {
				status := extractStatusFromFirstArg(pkg, node)
				if status > 0 {
					errs = append(errs, discoveredError{
						Status: status,
						Title:  http.StatusText(status),
					})
					return true
				}
				errs = append(errs, followCall(pkgPath, funcName, allPkgs, visited, func(p *packages.Package, b *ast.BlockStmt, v map[string]bool) []discoveredError {
					return discoverErrors(p, b, allPkgs, v, errType, statusField)
				})...)
				return true
			}

			if callResultSatisfiesErrorInterface(pkg, node) {
				status := extractStatusFromFirstArg(pkg, node)
				if status > 0 {
					errs = append(errs, discoveredError{
						Status: status,
						Title:  http.StatusText(status),
					})
					return true
				}
			}

			if pkgPath != "" && isInModule(pkgPath, pkg.PkgPath) {
				errs = append(errs, followCall(pkgPath, funcName, allPkgs, visited, func(p *packages.Package, b *ast.BlockStmt, v map[string]bool) []discoveredError {
					return discoverErrors(p, b, allPkgs, v, errType, statusField)
				})...)
			}
		}

		return true
	})

	errs = append(errs, discoverErrorVars(pkg, body, allPkgs, errType, statusField)...)

	return errs
}

func discoverErrorVars(pkg *packages.Package, body *ast.BlockStmt, allPkgs map[string]*packages.Package, errType *errorTypeInfo, statusField string) []discoveredError {
	if pkg.TypesInfo == nil {
		return nil
	}

	var errs []discoveredError
	seen := map[string]bool{}

	ast.Inspect(body, func(n ast.Node) bool {
		var obj types.Object

		switch node := n.(type) {
		case *ast.Ident:
			if use, ok := pkg.TypesInfo.Uses[node]; ok {
				obj = use
			}
		case *ast.SelectorExpr:
			if use, ok := pkg.TypesInfo.Uses[node.Sel]; ok {
				obj = use
			}
		}

		if obj == nil {
			return true
		}
		v, ok := obj.(*types.Var)
		if !ok || v.Parent() == nil {
			return true
		}
		if v.Parent() != v.Pkg().Scope() {
			return true
		}

		info := typeInfoFromTypesType(v.Type())
		if info == nil || info.PkgPath != errType.PkgPath || info.TypeName != errType.TypeName {
			return true
		}

		key := v.Pkg().Path() + "." + v.Name()
		if seen[key] {
			return true
		}
		seen[key] = true

		status := resolveVarStatus(v, allPkgs, errType, statusField)
		if status > 0 {
			errs = append(errs, discoveredError{
				Status: status,
				Title:  http.StatusText(status),
			})
		}

		return true
	})

	return errs
}

func extractStatusFromExpr(pkg *packages.Package, expr ast.Expr, allPkgs map[string]*packages.Package, errType *errorTypeInfo, statusField string) int {
	switch e := expr.(type) {
	case *ast.UnaryExpr:
		if e.Op == token.AND {
			if comp, ok := e.X.(*ast.CompositeLit); ok {
				return extractStatusFromComposite(pkg, comp, statusField)
			}
		}
	case *ast.CompositeLit:
		return extractStatusFromComposite(pkg, e, statusField)
	case *ast.CallExpr:
		status := extractStatusFromFirstArg(pkg, e)
		if status > 0 {
			return status
		}
		funcName, pkgPath := resolveCallTarget(pkg, e)
		if funcName != "" {
			visited := map[string]bool{}
			errs := followCall(pkgPath, funcName, allPkgs, visited, func(p *packages.Package, b *ast.BlockStmt, v map[string]bool) []discoveredError {
				return discoverErrors(p, b, allPkgs, v, errType, statusField)
			})
			if len(errs) > 0 {
				return errs[0].Status
			}
		}
	}
	return 0
}

func followCall(pkgPath, funcName string, allPkgs map[string]*packages.Package, visited map[string]bool, discover func(*packages.Package, *ast.BlockStmt, map[string]bool) []discoveredError) []discoveredError {
	key := pkgPath + "." + funcName
	if visited[key] {
		return nil
	}
	visited[key] = true

	calledPkg, ok := allPkgs[pkgPath]
	if !ok {
		return nil
	}

	var errs []discoveredError
	for _, file := range calledPkg.Syntax {
		ast.Inspect(file, func(nn ast.Node) bool {
			fn, ok := nn.(*ast.FuncDecl)
			if !ok || fn.Name == nil || fn.Name.Name != funcName {
				return true
			}
			errs = append(errs, discover(calledPkg, fn.Body, visited)...)
			return false
		})
	}
	return errs
}

func resolveVarStatus(v *types.Var, allPkgs map[string]*packages.Package, errType *errorTypeInfo, statusField string) int {
	pkg, ok := allPkgs[v.Pkg().Path()]
	if !ok {
		return 0
	}

	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.VAR {
				continue
			}
			for _, spec := range gen.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, name := range vs.Names {
					if name.Name != v.Name() || i >= len(vs.Values) {
						continue
					}
					return extractStatusFromExpr(pkg, vs.Values[i], allPkgs, errType, statusField)
				}
			}
		}
	}

	return 0
}
