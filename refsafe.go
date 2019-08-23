package refsafe

import (
	"github.com/Matts966/refsafe/analysisutil"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
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

// var shouldBeCalledBefore = [][2]string{
// 	{canAddr, addr},
// }

// func getMethod(name string) *types.Func {
// 	rep := regexp.MustCompile(`^(\(.*\))?\.(.*)$`)
// 	name = rep.ReplaceAllString()
// }

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
	// canAddr := analysisutil.MethodOf(val, "reflect.CanAddr")
	// addr := analysisutil.MethodOf(val, "reflect.Addr")

	funcs := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs
	// TODO(Matts966): Check the code depends on reflect, and early return if not.
	for _, f := range funcs {
		for _, b := range f.Blocks {
			for i, instr := range b.Instrs {
				for f, c := range funcToCan {
					m := analysisutil.MethodOf(val, f)
					cm := analysisutil.MethodOf(val, c)

					// pass.Reportf(instr.Pos(), "%#v, %#v", m, cm)

					if !analysisutil.Called(instr, nil, m) {
						continue
					}

					called, ok := analysisutil.CalledFromBefore(b, i, val, cm)
					// pass.Reportf(instr.Pos(), "%#v, %#v", called, ok)
					if called && ok {
						
						continue
					}
					pass.Reportf(instr.Pos(), c+" should be called before calling "+f)
				}
			}
		}
	}
	return nil, nil
}
