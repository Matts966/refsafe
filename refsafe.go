package refsafe

import (
	"go/token"

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
	canAddr = "(reflect.Value).CanAddr"
	addr = "(reflect.Value).Addr"
)

var valAsgnPos map[string]token.Pos

func run(pass *analysis.Pass) (interface{}, error) {
	funcs := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA).SrcFuncs
	// TODO(Matts966): Check the code depends on reflect, and early return if not.
	for _, f := range funcs {
		for _, b := range f.Blocks {
			seen := make([]bool, len(f.Blocks))
			seen[b.Index] = true
			calledFuncs := make(map[string]struct{})
			for _, instr := range b.Instrs {
				v, ok := instr.(ssa.Value)
				if !ok {
					continue
				}
				ok, name := getFuncNameIfCall(v)
				if !ok {
					continue
				}
				calledFuncs[name] = struct{}{}
				if name != addr {
					continue
				}
				if _, ok := calledFuncs[canAddr]; ok {
					continue
				}
				for _, p := range b.Preds {
					s2 := make([]bool, len(seen))
					copy(s2, seen)
					if recCalledByAllPred(canAddr, p, s2) {
						continue
					}
				}
				
				pass.Reportf(instr.Pos(), "reflect.CanAddr should be called before calling reflect.Addr")
			}
		}
	}
	return nil, nil
}

func recCalledByAllPred(name string, pred *ssa.BasicBlock, seen []bool) bool {
	if seen[pred.Index] {
		return false
	}
	seen[pred.Index] = true
	for _, pi := range pred.Instrs {
		v, ok := pi.(ssa.Value)
		if !ok {
			continue
		}
		ok, fn := getFuncNameIfCall(v)
		if !ok {
			continue
		}
		if fn == name {
			return true
		}
	}
	for _, p := range pred.Preds {
		s2 := make([]bool, len(seen))
		copy(s2, seen)
		if recCalledByAllPred(name, p, s2) {
			return false
		}
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
