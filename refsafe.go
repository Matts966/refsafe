package refsafe

import (
	"strconv"

	"github.com/Matts966/refsafe/analysisutil"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/ssa"
)

var Analyzer = &analysis.Analyzer{
	Name: "refsafe",
	Doc:  Doc,
	Run:  run,
	Requires: []*analysis.Analyzer{
		buildssa.Analyzer,
	},
}

const (
	Doc          = "Refsafe is a static analysis tool for using reflect package safely."
	canAddr      = "reflect.CanAddr"
	addr         = "reflect.Addr"
	canInterface = "reflect.CanInterface"
	getInterface = "reflect.Interface"
)

var funcToCan = map[string]string{
	"Addr":      "CanAddr",
	"Interface": "CanInterface",
	"Set":       "CanSet",
}

func run(pass *analysis.Pass) (interface{}, error) {
	val := analysisutil.TypeOf(pass, "reflect", "Value")

	setPointer := analysisutil.MethodOf(val, "SetPointer")
	kind := analysisutil.MethodOf(val, "Kind")
	up, err := analysisutil.LookupFromImportString("reflect", "UnsafePointer")
	if err != nil {
		return nil, err
	}

	funcs := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs
	for _, f := range funcs {
		if reflectNotUsed(pass, f) {
			continue
		}

		for _, b := range f.Blocks {
			for i, instr := range b.Instrs {
				for f, c := range funcToCan {
					m := analysisutil.MethodOf(val, f)
					cm := analysisutil.MethodOf(val, c)
					if !Called(instr, nil, m) {
						continue
					}

					callI, ok := instr.(ssa.CallInstruction)
					if !ok {
						continue
					}

					called, ok := analysisutil.CalledFromBefore(b, i, callI.Common().Args[0], cm)
					if called && ok {
						continue
					}
					pass.Reportf(instr.Pos(), c+" should be called before calling "+f)
				}

				recv := analysisutil.ReturnReceiverIfCalled(instr, setPointer)
				if recv == nil {
					continue
				}
				if analysisutil.CalledBeforeAndComparedTo(b, recv, kind, up) {
					continue
				}
				pass.Reportf(instr.Pos(), "Kind should be UnsafePointer when calling SetPointer")
			}
		}
	}
	return nil, nil
}

func reflectNotUsed(pass *analysis.Pass, f *ssa.Function) bool {
	if f == nil {
		return true
	}
	fo := f.Object()
	if fo == nil {
		return true
	}
	ff := analysisutil.File(pass, fo.Pos())
	if ff == nil {
		return true
	}
	for _, i := range ff.Imports {
		path, err := strconv.Unquote(i.Path.Value)
		if err != nil {
			continue
		}
		if path == "reflect" {
			return false
		}
	}
	return true
}
