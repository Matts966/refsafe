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
	st                 types.Type
	open               *types.Func
	close              *types.Func
	doSomething        *types.Func
	doSomethingSpecial *types.Func
	errFunc            *types.Func
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
	doSomethingSpecial = analysisutil.MethodOf(st, "a.doSomethingSpecial")
	errFunc = analysisutil.MethodOf(st, "a.err")
	ie, err := analysisutil.LookupFromImportString("io", "EOF")
	if err != nil {
		return nil, err
	}

	funcs := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs
	for _, f := range funcs {
		for _, b := range f.Blocks {
			for i, instr := range b.Instrs {
				recv := analysisutil.ReturnReceiverIfCalled(instr, doSomething)
				if recv == nil {
					continue
				}
				if called, ok := analysisutil.CalledFromAfter(b, i, recv, close); !(called && ok) {
					pass.Reportf(instr.Pos(), "close should be called after calling doSomething")
				}
				if called, ok := analysisutil.CalledFromBefore(b, i, recv, open); !(called && ok) {
					pass.Reportf(instr.Pos(), "open should be called before calling doSomething")
				}
			}

			for i, instr := range b.Instrs {
				recv := analysisutil.ReturnReceiverIfCalled(instr, doSomethingSpecial)
				if recv == nil {
					continue
				}
				if called, ok := analysisutil.CalledFromBefore(b, i, recv, errFunc); !(called && ok) {
					pass.Reportf(instr.Pos(), "err not called")
				}
				if analysisutil.CalledBeforeAndEqualTo(b, recv, errFunc, ie) {
					continue
				}
				pass.Reportf(instr.Pos(), "err should be io.EOF when calling doSomethingSpecial")
			}
		}
	}

	return nil, nil
}
