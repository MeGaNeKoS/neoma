package main

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/packages"
)

func callReturnsErrorType(pkg *packages.Package, call *ast.CallExpr, errType *errorTypeInfo) bool {
	if pkg.TypesInfo == nil {
		return false
	}
	tv, ok := pkg.TypesInfo.Types[call]
	if !ok {
		return false
	}
	info := typeInfoFromTypesType(tv.Type)
	if info != nil && info.PkgPath == errType.PkgPath && info.TypeName == errType.TypeName {
		return true
	}
	if tup, ok := tv.Type.(*types.Tuple); ok {
		for i := range tup.Len() {
			info := typeInfoFromTypesType(tup.At(i).Type())
			if info != nil && info.PkgPath == errType.PkgPath && info.TypeName == errType.TypeName {
				return true
			}
		}
	}
	return false
}

func callResultSatisfiesErrorInterface(pkg *packages.Package, call *ast.CallExpr) bool {
	if pkg.TypesInfo == nil {
		return false
	}
	tv, ok := pkg.TypesInfo.Types[call]
	if !ok {
		return false
	}
	return typeHasStatusCodeMethod(tv.Type)
}

func findStatusCodeType(allPkgs map[string]*packages.Package) *errorTypeInfo {
	for _, pkg := range allPkgs {
		if pkg.Types == nil {
			continue
		}
		scope := pkg.Types.Scope()
		for _, name := range scope.Names() {
			obj := scope.Lookup(name)
			tn, ok := obj.(*types.TypeName)
			if !ok {
				continue
			}
			named, ok := tn.Type().(*types.Named)
			if !ok {
				continue
			}
			ptrType := types.NewPointer(named)
			if !typeHasStatusCodeMethod(ptrType) {
				continue
			}
			if tn.Pkg() == nil {
				continue
			}
			return &errorTypeInfo{
				PkgPath:  tn.Pkg().Path(),
				TypeName: tn.Name(),
			}
		}
	}
	return nil
}

func findStatusFieldName(errType *errorTypeInfo, allPkgs map[string]*packages.Package) string {
	pkg, ok := allPkgs[errType.PkgPath]
	if !ok {
		return "Status"
	}

	statusMethodName := findStatusMethodName(errType, pkg)
	if statusMethodName == "" {
		return "Status"
	}

	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name == nil || fn.Name.Name != statusMethodName {
				continue
			}
			if fn.Recv == nil || len(fn.Recv.List) == 0 {
				continue
			}
			recvType := fn.Recv.List[0].Type
			if star, ok := recvType.(*ast.StarExpr); ok {
				recvType = star.X
			}
			var recvName string
			switch rt := recvType.(type) {
			case *ast.Ident:
				recvName = rt.Name
			case *ast.SelectorExpr:
				recvName = rt.Sel.Name
			default:
				continue
			}
			if recvName != errType.TypeName {
				continue
			}

			if fn.Body == nil {
				continue
			}
			for _, stmt := range fn.Body.List {
				ret, ok := stmt.(*ast.ReturnStmt)
				if !ok || len(ret.Results) != 1 {
					continue
				}
				sel, ok := ret.Results[0].(*ast.SelectorExpr)
				if !ok {
					continue
				}
				if _, ok := sel.X.(*ast.Ident); ok {
					return sel.Sel.Name
				}
			}
		}
	}

	return "Status"
}

func findStatusMethodName(errType *errorTypeInfo, pkg *packages.Package) string {
	if pkg.Types == nil {
		return ""
	}
	obj := pkg.Types.Scope().Lookup(errType.TypeName)
	if obj == nil {
		return ""
	}
	tn, ok := obj.(*types.TypeName)
	if !ok {
		return ""
	}
	named, ok := tn.Type().(*types.Named)
	if !ok {
		return ""
	}
	ptrType := types.NewPointer(named)
	mset := types.NewMethodSet(ptrType)
	for i := range mset.Len() {
		sel := mset.At(i)
		fn, ok := sel.Obj().(*types.Func)
		if !ok {
			continue
		}
		if isStatusMethodSignature(fn) {
			return fn.Name()
		}
	}
	return ""
}

func isErrorInterface(t types.Type) bool {
	var iface *types.Interface
	switch u := t.(type) {
	case *types.Interface:
		iface = u
	case *types.Named:
		var ok bool
		iface, ok = u.Underlying().(*types.Interface)
		if !ok {
			return false
		}
	default:
		return false
	}
	if iface.NumMethods() != 1 {
		return false
	}
	m := iface.Method(0)
	sig, ok := m.Type().(*types.Signature)
	if !ok {
		return false
	}
	if sig.Params().Len() != 0 || sig.Results().Len() != 1 {
		return false
	}
	basic, ok := sig.Results().At(0).Type().(*types.Basic)
	return ok && basic.Kind() == types.String
}

func isErrorMethodSignature(fn *types.Func) bool {
	sig, ok := fn.Type().(*types.Signature)
	if !ok {
		return false
	}
	if sig.Params().Len() != 0 || sig.Results().Len() != 1 {
		return false
	}
	basic, ok := sig.Results().At(0).Type().(*types.Basic)
	return ok && basic.Kind() == types.String
}

func isStatusMethodSignature(fn *types.Func) bool {
	sig, ok := fn.Type().(*types.Signature)
	if !ok {
		return false
	}
	if sig.Params().Len() != 0 || sig.Results().Len() != 1 {
		return false
	}
	basic, ok := sig.Results().At(0).Type().(*types.Basic)
	return ok && basic.Kind() == types.Int
}

func returnsErrorType(pkgPath, funcName string, allPkgs map[string]*packages.Package, errType *errorTypeInfo) bool {
	pkg, ok := allPkgs[pkgPath]
	if !ok || pkg.TypesInfo == nil {
		return false
	}

	obj := pkg.Types.Scope().Lookup(funcName)
	if obj == nil {
		return false
	}
	fn, ok := obj.(*types.Func)
	if !ok {
		return false
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok {
		return false
	}

	results := sig.Results()
	for i := range results.Len() {
		info := typeInfoFromTypesType(results.At(i).Type())
		if info != nil && info.PkgPath == errType.PkgPath && info.TypeName == errType.TypeName {
			return true
		}
	}

	return false
}

func typeHasStatusCodeMethod(t types.Type) bool {
	mset := types.NewMethodSet(t)
	hasIntMethod := false
	hasErrorMethod := false
	for i := range mset.Len() {
		fn, ok := mset.At(i).Obj().(*types.Func)
		if !ok {
			continue
		}
		if isStatusMethodSignature(fn) {
			hasIntMethod = true
		}
		if isErrorMethodSignature(fn) {
			hasErrorMethod = true
		}
		if hasIntMethod && hasErrorMethod {
			return true
		}
	}
	return false
}
