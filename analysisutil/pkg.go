package analysisutil

import (
	"go/types"
	"strings"
	"golang.org/x/tools/go/loader"
)

// RemoVendor removes vendoring infomation from import path.
func RemoveVendor(path string) string {
	i := strings.Index(path, "vendor")
	if i >= 0 {
		return path[i+len("vendor")+1:]
	}
	return path
}

// LookupFromImports finds an object from import paths.
func LookupFromImports(imports []*types.Package, path, name string) types.Object {
	path = RemoveVendor(path)
	for i := range imports {
		if path == RemoveVendor(imports[i].Path()) {
			return imports[i].Scope().Lookup(name)
		}
	}
	return nil
}

// LookupFromImportString finds an object from package and name.
func LookupFromImportString(importPkg string, name string) (types.Object, error) {
	lc := loader.Config{}
	lc.Import(importPkg)
	p, err := lc.Load()
	if err != nil {
		return nil, err
	}
	return p.Package(importPkg).Pkg.Scope().Lookup(name), nil
}
