package analysisutil_test

import (
	"go/types"
	"testing"

	"github.com/Matts966/refsafe/analysisutil"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
	"golang.org/x/tools/go/analysis/passes/buildssa"
)

var (
	st          types.Type
	open        *types.Func
	close       *types.Func
	doSomething *types.Func
)

var Analyzer = &analysis.Analyzer{
	Name: "test_call",
	Run:  run,
	Requires: []*analysis.Analyzer{
		buildssa.Analyzer,
	},
}

func Test(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, Analyzer, "a")
}

func run(pass *analysis.Pass) (interface{}, error) {
	st = analysisutil.LookupFromImports([]*types.Package{
		pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).Pkg.Pkg,
	}, "a", "st").Type().(*types.Named)
	open = analysisutil.MethodOf(st, "a.open")
	close = analysisutil.MethodOf(st, "a.close")
	doSomething = analysisutil.MethodOf(st, "a.doSomething")
	funcs := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs
	for _, f := range funcs {
		for _, b := range f.Blocks {
			for i, instr := range b.Instrs {
				if !analysisutil.Called(instr, nil, doSomething) {
					continue
				}

				if called, ok := analysisutil.CalledFromAfter(b, i, st, close); !(called && ok) {
					pass.Reportf(instr.Pos(), "close should be called after calling doSomething")
				}

				// log.Println("ここから", open)
				if called, ok := analysisutil.CalledFromBefore(b, i, st, open); !(called && ok) {
					pass.Reportf(instr.Pos(), "open should be called before calling doSomething")
				}
				// log.Println("ここまで", open)
			}
		}
	}
	return nil, nil
}
