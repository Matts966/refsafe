package refsafe

import (
	"go/types"
	"strconv"

	"github.com/Matts966/refsafe/analysisutil"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/ssa"
)

var Analyzer = &analysis.Analyzer{
	Name: "refsafe",
	Doc:  "Refsafe is a static analysis tool for using reflect package safely.",
	Run:  run,
	Requires: []*analysis.Analyzer{
		buildssa.Analyzer,
	},
}

var funcToCanFunc = map[*types.Func]*types.Func{}
var funcNameToCanName = map[string]string{
	"Addr":      "CanAddr",
	"Interface": "CanInterface",
	"Set":       "CanSet",
}

var funcToKind = map[*types.Func]types.Object{}
var funcNameToKindName = map[string]string{
	"SetPointer": "UnsafePointer",
	"SetBool":    "Bool",
	"SetString":  "String",
}

func run(pass *analysis.Pass) (interface{}, error) {
	val := analysisutil.TypeOf(pass, "reflect", "Value")
	for f, c := range funcNameToCanName {
		ff := analysisutil.MethodOf(val, f)
		cf := analysisutil.MethodOf(val, c)
		funcToCanFunc[ff] = cf
	}

	kind := analysisutil.MethodOf(val, "Kind")
	for f, c := range funcNameToKindName {
		ff := analysisutil.MethodOf(val, f)
		co, err := analysisutil.LookupFromImportString("reflect", c)
		if err != nil {
			return nil, err
		}
		funcToKind[ff] = co
	}

	funcs := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs
	for _, f := range funcs {
		if reflectNotUsed(pass, f) {
			continue
		}

		for _, b := range f.Blocks {
			for i, instr := range b.Instrs {
				for f, c := range funcToCanFunc {
					if !Called(instr, nil, f) {
						continue
					}

					callI, ok := instr.(ssa.CallInstruction)
					if !ok {
						continue
					}

					called, ok := analysisutil.CalledFromBefore(b, i, callI.Common().Args[0], c)
					if called && ok {
						continue
					}

					pass.Reportf(instr.Pos(), c.Name()+" should be called before calling "+f.Name())
				}

				for f, k := range funcToKind {
					recv := analysisutil.ReturnReceiverIfCalled(instr, f)
					if recv == nil {
						continue
					}
					if analysisutil.CalledBeforeAndComparedTo(b, recv, kind, k) {
						continue
					}
					pass.Reportf(instr.Pos(), "Kind should be "+k.Name()+" when calling "+f.Name())
				}
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
