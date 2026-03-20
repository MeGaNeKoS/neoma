package main

import (
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

func compositeTypeInfo(pkg *packages.Package, comp *ast.CompositeLit) *errorTypeInfo {
	if pkg.TypesInfo == nil {
		return nil
	}
	tv, ok := pkg.TypesInfo.Types[comp]
	if !ok {
		return nil
	}
	return typeInfoFromTypesType(tv.Type)
}

func constantInt(s string) (int, bool) {
	v, err := strconv.Atoi(s)
	return v, err == nil
}

func extractStatusFromComposite(pkg *packages.Package, comp *ast.CompositeLit, statusField string) int {
	for _, elt := range comp.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		ident, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}
		if ident.Name != statusField {
			continue
		}
		if lit, ok := kv.Value.(*ast.BasicLit); ok && lit.Kind == token.INT {
			v, err := strconv.Atoi(lit.Value)
			if err == nil {
				return v
			}
		}
		if pkg.TypesInfo != nil {
			if tv, ok := pkg.TypesInfo.Types[kv.Value]; ok {
				if tv.Value != nil {
					if v, exact := constantInt(tv.Value.String()); exact {
						return v
					}
				}
			}
		}
	}
	return 0
}

func extractStatusFromFirstArg(pkg *packages.Package, call *ast.CallExpr) int {
	if len(call.Args) == 0 {
		return 0
	}
	if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.INT {
		v, err := strconv.Atoi(lit.Value)
		if err == nil {
			return v
		}
	}
	if pkg.TypesInfo != nil {
		if tv, ok := pkg.TypesInfo.Types[call.Args[0]]; ok && tv.Value != nil {
			if v, exact := constantInt(tv.Value.String()); exact {
				return v
			}
		}
	}
	return 0
}

func isInModule(pkgPath, refPkg string) bool {
	parts := strings.SplitN(refPkg, "/", 4)
	if len(parts) < 3 {
		return strings.HasPrefix(pkgPath, refPkg)
	}
	moduleRoot := strings.Join(parts[:3], "/")
	return strings.HasPrefix(pkgPath, moduleRoot)
}

func resolveCallTarget(pkg *packages.Package, call *ast.CallExpr) (funcName, pkgPath string) {
	switch fn := call.Fun.(type) {
	case *ast.SelectorExpr:
		funcName = fn.Sel.Name

		if pkg.TypesInfo != nil {
			if use, ok := pkg.TypesInfo.Uses[fn.Sel]; ok {
				if f, ok := use.(*types.Func); ok {
					if f.Pkg() != nil {
						pkgPath = f.Pkg().Path()
					}
				}
			}
		}

		if pkgPath == "" {
			if ident, ok := fn.X.(*ast.Ident); ok && pkg.TypesInfo != nil {
				if obj, ok := pkg.TypesInfo.Uses[ident]; ok {
					if pkgName, ok := obj.(*types.PkgName); ok {
						pkgPath = pkgName.Imported().Path()
					}
				}
			}
		}

	case *ast.Ident:
		funcName = fn.Name
		pkgPath = pkg.PkgPath
	}

	return
}
