package refsafe

import (
	"fmt"
	"go/types"

	"github.com/gostaticanalysis/analysisutil"
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
	Doc = "Refsafe is a static analysis tool for using reflect package safely."
	// canAddr = "(reflect.Value).CanAddr"
	// addr    = "(reflect.Value).Addr"
)

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
	canAddr := analysisutil.MethodOf(val, "CanAddr")
	addr := analysisutil.MethodOf(val, "Addr")

	funcs := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs
	// TODO(Matts966): Check the code depends on reflect, and early return if not.
	for _, f := range funcs {
		for _, b := range f.Blocks {
			for i, instr := range b.Instrs {
				fn := getFunction(instr)
				if fn == nil {
					continue
				}
				if fn.f != addr {
					continue
				}
				cf := &function{
					fn.recv,
					canAddr,
				}
				if cf.calledBeforeInBlock(b, i) {
					continue
				}
				visited := make([]bool, len(f.Blocks))
				visited[b.Index] = true
				//pass.Reportf(instr.Pos(), "%#v", fn.recv)
				if cf.calledByAllPreds(b.Preds, visited) {
					continue
				}
				pass.Reportf(instr.Pos(), "(reflect.Value).CanAddr should be true when invoking (reflect.Value).Addr")
			}
		}
	}
	return nil, nil
}

func getFunction(i ssa.Instruction) *function {
	ci, ok := i.(ssa.CallInstruction)
	if !ok {
		return nil
	}
	cm := ci.Common()
	if cm == nil {
		return nil
	}
	if cm.Method != nil {
		return &function{
			cm.Args[0],
			cm.Method,
		}
	}
	cl := cm.StaticCallee()
	if cl == nil {
		return nil
	}
	fn, ok := cl.Object().(*types.Func)
	if !ok {
		return nil
	}
	if cm.Signature().Recv() != nil {
		return &function{
			cm.Args[0],
			fn,
		}
	}
	return &function{
		nil,
		fn,
	}
}

type function struct {
	recv ssa.Value
	f    *types.Func
}

func (f function) same(i ssa.Instruction) bool {
	fn := getFunction(i)
	if fn == nil {
		return false
	}
	if fn.f != f.f {
		return false
	}
	// panic(fmt.Errorf("\n\n%#v\n%#v\n\n", fn.recv, f.recv))
	if fn.recv != nil || f.recv != nil {
		return false
	}

	refs := fn.recv.Referrers()
	if refs == nil {
		return false
	}
	for _, ref := range *refs {
		ref
	}

	refs := f.recv.Referrers()
	if refs == nil {
		return false
	}
	for _, ref := range *refs {
		
	}

	return false
}

func (f function) calledBeforeInBlock(b *ssa.BasicBlock, nowIndex int) bool {
	for i := 0; i < nowIndex; i++ {
		if f.same(b.Instrs[i]) {
			return true
		}
	}
	return false
}

func (f function) calledByAllPreds(preds []*ssa.BasicBlock, visited []bool) bool {
	for _, p := range preds {
		var calledInThisBlock bool
		for _, pi := range p.Instrs {
			if f.same(pi) {
				calledInThisBlock = true
				break
			}
		}
		if calledInThisBlock {
			continue
		}

		v2 := visited
		if v2[p.Index] {
			return false
		}
		v2[p.Index] = true

		if f.calledByAllPreds(p.Preds, v2) {
			continue
		}
		return false
	}
	return true
}

func getFuncNameIfCall(val ssa.Value) (bool, string) {
	c, ok := val.(*ssa.Call)
	if !ok {
		return ok, ""
	}
	return ok, c.Common().Value.String()
}
