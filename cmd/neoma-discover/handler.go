package main

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"
)

func concreteTypeFromAST(method *types.Func, allPkgs map[string]*packages.Package) *errorTypeInfo {
	pkgPath := method.Pkg().Path()
	methodName := method.Name()

	pkg, ok := allPkgs[pkgPath]
	if !ok {
		return nil
	}

	for _, file := range pkg.Syntax {
		var result *errorTypeInfo
		ast.Inspect(file, func(n ast.Node) bool {
			if result != nil {
				return false
			}
			fn, ok := n.(*ast.FuncDecl)
			if !ok {
				return true
			}
			if fn.Name == nil || fn.Name.Name != methodName {
				return true
			}
			if fn.Recv == nil || len(fn.Recv.List) == 0 {
				return true
			}
			if pkg.TypesInfo != nil {
				if def, ok := pkg.TypesInfo.Defs[fn.Name]; ok {
					if def != method {
						return true
					}
				}
			}

			ast.Inspect(fn.Body, func(nn ast.Node) bool {
				if result != nil {
					return false
				}
				ret, ok := nn.(*ast.ReturnStmt)
				if !ok {
					return true
				}
				for _, expr := range ret.Results {
					info := typeInfoFromExpr(pkg, expr)
					if info != nil {
						result = info
						return false
					}
				}
				return true
			})
			return false
		})
		if result != nil {
			return result
		}
	}

	return nil
}

func findHandlerType(pkgs []*packages.Package) *handlerTypeInfo {
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			var found *handlerTypeInfo
			ast.Inspect(file, func(n ast.Node) bool {
				if found != nil {
					return false
				}
				assign, ok := n.(*ast.AssignStmt)
				if !ok {
					return true
				}
				for i := range assign.Lhs {
					if i >= len(assign.Rhs) {
						continue
					}
					if pkg.TypesInfo == nil {
						continue
					}
					tv, ok := pkg.TypesInfo.Types[assign.Rhs[i]]
					if !ok {
						continue
					}
					if !satisfiesErrorHandler(tv.Type) {
						continue
					}
					info := handlerTypeInfoFromType(tv.Type)
					if info != nil {
						found = info
					}
				}
				return found == nil
			})
			if found != nil {
				return found
			}
		}
	}
	return nil
}

func findNewErrorReturnType(handlerType types.Type, allPkgs map[string]*packages.Package) *errorTypeInfo {
	ptr, ok := handlerType.(*types.Pointer)
	if !ok {
		return nil
	}
	named, ok := ptr.Elem().(*types.Named)
	if !ok {
		return nil
	}

	for i := range named.NumMethods() {
		m := named.Method(i)
		if matchesErrorConstructorSignature(m) {
			return concreteTypeFromAST(m, allPkgs)
		}
	}
	return nil
}

func handlerTypeInfoFromType(t types.Type) *handlerTypeInfo {
	raw := t
	ptr, ok := t.(*types.Pointer)
	if ok {
		t = ptr.Elem()
	}
	named, ok := t.(*types.Named)
	if !ok {
		return nil
	}
	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil {
		return nil
	}
	return &handlerTypeInfo{
		PkgPath:  obj.Pkg().Path(),
		TypeName: obj.Name(),
		rawType:  raw,
	}
}

func hasErrorConstructorMethod(mset *types.MethodSet) bool {
	for i := range mset.Len() {
		obj := mset.At(i).Obj()
		fn, ok := obj.(*types.Func)
		if !ok {
			continue
		}
		if matchesErrorConstructorSignature(fn) {
			return true
		}
	}
	return false
}

func matchesErrorConstructorSignature(fn *types.Func) bool {
	sig, ok := fn.Type().(*types.Signature)
	if !ok {
		return false
	}
	params := sig.Params()
	if params.Len() != 3 {
		return false
	}
	p0Basic, ok := params.At(0).Type().(*types.Basic)
	if !ok || p0Basic.Kind() != types.Int {
		return false
	}
	p1Basic, ok := params.At(1).Type().(*types.Basic)
	if !ok || p1Basic.Kind() != types.String {
		return false
	}
	if !sig.Variadic() {
		return false
	}
	p2Slice, ok := params.At(2).Type().(*types.Slice)
	if !ok {
		return false
	}
	if !isErrorInterface(p2Slice.Elem()) {
		return false
	}
	results := sig.Results()
	if results.Len() < 1 {
		return false
	}
	return typeHasStatusCodeMethod(results.At(0).Type())
}

func satisfiesErrorHandler(t types.Type) bool {
	mset := types.NewMethodSet(t)
	if hasErrorConstructorMethod(mset) {
		return true
	}
	if _, isPtr := t.(*types.Pointer); !isPtr {
		ptrMset := types.NewMethodSet(types.NewPointer(t))
		if hasErrorConstructorMethod(ptrMset) {
			return true
		}
	}
	return false
}

func typeInfoFromExpr(pkg *packages.Package, expr ast.Expr) *errorTypeInfo {
	unary, ok := expr.(*ast.UnaryExpr)
	if ok && unary.Op == token.AND {
		expr = unary.X
	}

	comp, ok := expr.(*ast.CompositeLit)
	if !ok {
		return nil
	}

	if pkg.TypesInfo == nil {
		return nil
	}

	tv, ok := pkg.TypesInfo.Types[comp]
	if !ok {
		return nil
	}

	return typeInfoFromTypesType(tv.Type)
}

func typeInfoFromTypesType(t types.Type) *errorTypeInfo {
	ptr, ok := t.(*types.Pointer)
	if ok {
		t = ptr.Elem()
	}
	named, ok := t.(*types.Named)
	if !ok {
		return nil
	}
	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil {
		return nil
	}
	return &errorTypeInfo{
		PkgPath:  obj.Pkg().Path(),
		TypeName: obj.Name(),
	}
}
