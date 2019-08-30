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
	// for _, sbcb := range shouldBeCalledBefore {
	// 	sbcb[0]
	// 	sbcb[1]
	// }
	// var reflect *types.Package
	// for _, ipkg := range pass.Pkg.Imports() {
	// 	if ipkg.Name() == "reflect" {
	// 		reflect = ipkg
	// 		break
	// 	}
	// }
	// if reflect.Name() != "reflect" {
	// 	return nil, nil
	// }
	// fmt.Println(reflect.Scope())
	// reflect.Scope().Lookup("reflect.MakeChan")
	val := analysisutil.TypeOf(pass, "reflect", "Value")

	setPointer := analysisutil.MethodOf(val, "SetPointer")
	kind := analysisutil.MethodOf(val, "Kind")
	up, err := analysisutil.LookupFromImportString("reflect", "UnsafePointer")
	if err != nil {
		return nil, err
	}

	// canAddr := analysisutil.MethodOf(val, "reflect.CanAddr")
	// addr := analysisutil.MethodOf(val, "reflect.Addr")
	funcs := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs

	for _, f := range funcs {

		if reflectNotUsed(pass, f) {
			continue
		}
		// fmt.Println()
		// fmt.Printf("f: %v\n", f)

		for _, b := range f.Blocks {
			for i, instr := range b.Instrs {

				// fmt.Printf("instr: %#v\n", instr)
				// fmt.Printf("type: %T\n", instr)
				// fmt.Println()

				for f, c := range funcToCan {
					m := analysisutil.MethodOf(val, f)
					cm := analysisutil.MethodOf(val, c)

					// pass.Reportf(instr.Pos(), "%#v, %#v", m, cm)

					if !Called(instr, nil, m) {
						continue
					}

					callI, ok := instr.(ssa.CallInstruction)
					if !ok {
						continue
					}

					called, ok := analysisutil.CalledFromBefore(b, i, callI.Common().Args[0], cm)
					// pass.Reportf(instr.Pos(), "%#v, %#v", called, ok)
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
