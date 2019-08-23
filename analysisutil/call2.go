package analysisutil

import (
	"go/types"

	"golang.org/x/tools/go/ssa"
)

// FromBefore checks whether receiver's method is called in an instruction
// which belongs to before i-th instructions, or in succsor blocks of b.
// The first result is above value.
// The second result is whether type of i-th instruction does not much receiver
// or matches with ignore cases.
func (c *CalledChecker) FromBefore(b *ssa.BasicBlock, i int, receiver types.Type, methods ...*types.Func) (called, ok bool) {
	if b == nil || i < 0 || i >= len(b.Instrs) ||
		receiver == nil || len(methods) == 0 {
		return false, false
	}

	v, ok := b.Instrs[i].(ssa.Value)
	if !ok {
		return false, false
	}

	if !identical(v.Type(), receiver) {
		return false, false
	}

	from := &calledFrom{recv: v, fs: methods, ignore: c.Ignore}
	if from.ignoredBefore() {
		return false, false
	}

	if from.instrs(b.Instrs[:i]) ||
		from.preds(b) {
		return true, true
	}

	return false, true
}

func (c *calledFrom) ignoredBefore() bool {
	// refs := c.recv.Referrers()
	// if refs == nil {
	// 	return false
	// }

	// for _, ref := range *refs {
	// 	if !c.isOwn(ref) &&
	// 		((c.ignore != nil && c.ignore(ref)) ||
	// 			c.isRet(ref) || c.isArg(ref)) {
	// 		return true
	// 	}
	// }

	return false
}

func (c *calledFrom) preds(b *ssa.BasicBlock) bool {
	if c.done == nil {
		c.done = map[*ssa.BasicBlock]bool{}
	}

	if c.done[b] {
		return false
	}
	c.done[b] = true

	if len(b.Preds) == 0 {
		return false
	}

	for _, p := range b.Preds {
		if !c.instrs(p.Instrs) && !c.preds(p) {
			return false
		}
	}

	return true
}

// CalledFromBefore checks whether receiver's method is called in an instruction
// which belogns to before i-th instructions, or in succsor blocks of b.
// The first result is above value.
// The second result is whether type of i-th instruction does not much receiver
// or matches with ignore cases.
func CalledFromBefore(b *ssa.BasicBlock, i int, receiver types.Type, methods ...*types.Func) (called, ok bool) {
	return new(CalledChecker).FromBefore(b, i, receiver, methods...)
}
