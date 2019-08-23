package analysisutil_test

import (
	"testing"

	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"log"

	"github.com/Matts966/refsafe/analysisutil"

	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

var (
	ssapkg           *ssa.Package
	st               types.Type
	open             *types.Func
	close            *types.Func
	doSomthing       *types.Func
	beforeTestResult map[string]bool
	afterTestResult  map[string]bool
)

func init() {
	beforeTestResult = map[string]bool{
		"test1": false,
		"test2": true,
		"test3": true,
	}
	afterTestResult = map[string]bool{
		"test1": false,
		"test2": true,
		"test3": true,
	}

	fileName := "testdata/call/main.go"
	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, fileName, nil, parser.AllErrors)
	if err != nil {
		log.Fatalf("Error on parser.ParseFile: %v", err)
	}
	files := []*ast.File{f}

	ssapkg, _, err = ssautil.BuildPackage(
		&types.Config{Importer: importer.Default()},
		fset, types.NewPackage("main", ""), files,
		ssa.GlobalDebug,
	)
	if err != nil {
		log.Fatal(err)
	}

	st = analysisutil.LookupFromImports([]*types.Package{
		ssapkg.Pkg,
	}, "main", "st").Type().(*types.Named)
	open = analysisutil.MethodOf(st, "main.open")
	close = analysisutil.MethodOf(st, "main.close")
	doSomthing = analysisutil.MethodOf(st, "main.doSomething")
}

func TestCalledFrom(t *testing.T) {
	t.Parallel()
	for _, v := range ssapkg.Members {
		if f := ssapkg.Func(v.Name()); f != nil {
			for _, b := range f.Blocks {
				for ii, i := range b.Instrs {
					if !analysisutil.Called(i, nil, doSomthing) {
						continue
					}

					if called, ok := analysisutil.CalledFromAfter(b, ii, st, close); !(called && ok) {
						if !beforeTestResult[f.Name()] {
							continue
						}
					}

					if beforeTestResult[f.Name()] {
						continue
					}

					t.Fatal("Setup function not called")
				}
			}
		}
	}
}
func TestCalledFromBefore(t *testing.T) {
	t.Parallel()
	for _, v := range ssapkg.Members {
		if f := ssapkg.Func(v.Name()); f != nil {
			for _, b := range f.Blocks {
				for ii, i := range b.Instrs {
					if !analysisutil.Called(i, nil, close) {
						continue
					}

					if called, ok := analysisutil.CalledFromBefore(b, ii, st, open); !(called && ok) {
						if !afterTestResult[f.Name()] {
							continue
						}
					}

					if afterTestResult[f.Name()] {
						continue
					}

					t.Fatal("Setup function not called")
				}
			}
		}
	}
}
