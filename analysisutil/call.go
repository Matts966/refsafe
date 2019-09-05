package analysisutil

import (
	"go/token"
	"go/types"

	"golang.org/x/tools/go/ssa"
)

// CalledChecker checks a function is called.
// See From and Func.
type CalledChecker struct {
	Ignore func(instr ssa.Instruction) bool
}

func (c *CalledChecker) returnReceiverIfCalled(instr ssa.Instruction, f *types.Func) ssa.Value {
	call, ok := instr.(ssa.CallInstruction)
	if !ok {
		return nil
	}

	common := call.Common()
	if common == nil {
		return nil
	}

	callee := common.StaticCallee()
	if callee == nil {
		return nil
	}

	fn, ok := callee.Object().(*types.Func)
	if !ok {
		return nil
	}

	if fn != f {
		return nil
	}

	if len(common.Args) > 0 {
		return common.Args[0]
	}

	return nil
}

// Func returns true when f is called in the instr.
// If recv is not nil, Called also checks the receiver.
func (c *CalledChecker) Func(instr ssa.Instruction, recv ssa.Value, f *types.Func) bool {
	if c.Ignore != nil && c.Ignore(instr) {
		return false
	}

	call, ok := instr.(ssa.CallInstruction)
	if !ok {
		return false
	}

	common := call.Common()
	if common == nil {
		return false
	}

	callee := common.StaticCallee()
	if callee == nil {
		return false
	}

	fn, ok := callee.Object().(*types.Func)
	if !ok {
		return false
	}

	if recv != nil &&
		common.Signature().Recv() != nil &&
		(len(common.Args) == 0 && recv != nil || common.Args[0] != recv &&
			!isReferrer(recv, common.Args[0])) {
		return false
	}

	return fn == f
}

func isReferrer(a, b ssa.Value) bool {
	if a == nil || b == nil {
		return false
	}
	if a.Referrers() != nil {
		ars := *a.Referrers()

		for _, ar := range ars {
			arv, ok := ar.(ssa.Value)
			if !ok {
				continue
			}
			if arv == b {
				return true
			}

		}
	}

	if b.Referrers() != nil {
		brs := *b.Referrers()

		for _, br := range brs {
			brv, ok := br.(ssa.Value)
			if !ok {
				continue
			}
			if brv == a {
				return true
			}
		}
	}
	return false
}

// From checks whether receiver's method is called in an instruction
// which belogns to after i-th instructions, or in succsor blocks of b.
// The first result is above value.
// The second result is whether type of i-th instruction does not much receiver
// or matches with ignore cases.
func (c *CalledChecker) From(b *ssa.BasicBlock, i int, receiver types.Type, methods ...*types.Func) (called, ok bool) {
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
	if from.ignored() {
		return false, false
	}

	if from.instrs(b.Instrs[i+1:]) ||
		from.succs(b) {
		return true, true
	}

	return false, true
}

type calledFrom struct {
	recv   ssa.Value
	fs     []*types.Func
	done   map[*ssa.BasicBlock]bool
	ignore func(ssa.Instruction) bool
	start  *ssa.BasicBlock
}

func (c *calledFrom) ignored() bool {
	refs := c.recv.Referrers()
	if refs == nil {
		return false
	}

	for _, ref := range *refs {
		if !c.isOwn(ref) &&
			((c.ignore != nil && c.ignore(ref)) ||
				c.isRet(ref) || c.isArg(ref)) {
			return true
		}
	}

	return false
}

func (c *calledFrom) isOwn(instr ssa.Instruction) bool {
	v, ok := instr.(ssa.Value)
	if !ok {
		return false
	}
	return v == c.recv
}

func (c *calledFrom) isRet(instr ssa.Instruction) bool {

	ret, ok := instr.(*ssa.Return)
	if !ok {
		return false
	}

	for _, r := range ret.Results {
		if r == c.recv {
			return true
		}
	}

	return false
}

func (c *calledFrom) isArg(instr ssa.Instruction) bool {

	call, ok := instr.(ssa.CallInstruction)
	if !ok {
		return false
	}

	common := call.Common()
	if common == nil {
		return false
	}

	args := common.Args
	if common.Signature().Recv() != nil {
		args = args[1:]
	}

	for i := range args {
		if args[i] == c.recv {
			return true
		}
	}

	return false
}

func (c *calledFrom) instrs(instrs []ssa.Instruction) bool {
	for _, instr := range instrs {
		for _, f := range c.fs {
			// If pointer value is indirected, get the raw value.
			if ru, ok := c.recv.(*ssa.UnOp); ok && ru.Op == token.MUL {
				c.recv = ru.X
			}
			if Called(instr, c.recv, f) {
				return true
			}
		}
	}
	return false
}

func (c *calledFrom) calledIndex(instrs []ssa.Instruction) int {
	for i, instr := range instrs {
		for _, f := range c.fs {
			// If pointer value is indirected, get the raw value.
			if ru, ok := c.recv.(*ssa.UnOp); ok && ru.Op == token.MUL {
				c.recv = ru.X
			}
			if Called(instr, c.recv, f) {
				return i
			}
		}
	}
	return -1
}

func (c *calledFrom) succs(b *ssa.BasicBlock) bool {
	if c.done == nil {
		c.done = map[*ssa.BasicBlock]bool{}
	}

	if c.done[b] {
		return false
	}
	c.done[b] = true

	if len(b.Succs) == 0 {
		return false
	}

	for _, s := range b.Succs {
		if !c.instrs(s.Instrs) && !c.succs(s) {
			return false
		}
	}

	return true
}

// CalledFrom checks whether receiver's method is called in an instruction
// which belogns to after i-th instructions, or in succsor blocks of b.
// The first result is above value.
// The second result is whether type of i-th instruction does not much receiver
// or matches with ignore cases.
func CalledFrom(b *ssa.BasicBlock, i int, receiver types.Type, methods ...*types.Func) (called, ok bool) {
	return new(CalledChecker).From(b, i, receiver, methods...)
}

// Called returns true when f is called in the instr.
// If recv is not nil, Called also checks the receiver.
func Called(instr ssa.Instruction, recv ssa.Value, f *types.Func) bool {
	return new(CalledChecker).Func(instr, recv, f)
}

// ReturnReceiverIfCalled returns value of the first argment which is probablly the receiver
// when f is called in the instr. If recv is not nil, Called also checks the receiver.
func ReturnReceiverIfCalled(instr ssa.Instruction, f *types.Func) ssa.Value {
	return new(CalledChecker).returnReceiverIfCalled(instr, f)
}

// Before checks whether receiver's method is called in an instruction
// which belongs to before i-th instructions, or in preds blocks of b.
// The first result is above value.
// The second result is whether type of i-th instruction does not much receiver
// or matches with ignore cases.
func (c *CalledChecker) Before(b *ssa.BasicBlock, i int, receiver types.Type, methods ...*types.Func) (called, ok bool) {
	if b == nil || i < 0 || i >= len(b.Instrs) ||
		receiver == nil || len(methods) == 0 {
		return false, false
	}

	v, ok := b.Instrs[i].(ssa.Value)
	if !ok {
		return false, false
	}
	from := &calledFrom{recv: v, fs: methods, ignore: c.Ignore}

	if from.instrs(b.Instrs[:i]) ||
		from.preds(b) {
		return true, true
	}

	return false, true
}

// FromBefore checks whether receiver's method is called in instructions
// which belongs to before i-th instructions, or in preds blocks of b.
// The second result is whether type of i-th instruction does not much receiver
// or matches with ignore cases. FromBefore takes receiver as a value of ssa.Value.
func (c *CalledChecker) FromBefore(b *ssa.BasicBlock, i int, receiver ssa.Value, methods ...*types.Func) (called, ok bool) {
	if b == nil || i < 0 || i >= len(b.Instrs) ||
		receiver == nil || len(methods) == 0 {
		return false, false
	}

	// If pointer value is indirected, get the raw value.
	if ru, ok := receiver.(*ssa.UnOp); ok && ru.Op == token.MUL {
		receiver = ru.X
	}

	from := &calledFrom{recv: receiver, fs: methods, ignore: c.Ignore}

	if from.instrs(b.Instrs[:i]) || from.preds(b) {
		return true, true
	}
	return false, true
}

func (c *CalledChecker) FromAfter(b *ssa.BasicBlock, i int, receiver ssa.Value, methods ...*types.Func) (called, ok bool) {
	if b == nil || i < 0 || i >= len(b.Instrs) ||
		receiver == nil || len(methods) == 0 {
		return false, false
	}
	from := &calledFrom{recv: receiver, fs: methods, ignore: c.Ignore}

	if from.instrs(b.Instrs[i+1:]) || from.preds(b) {
		return true, true
	}
	return false, true
}

func (c *calledFrom) preds(b *ssa.BasicBlock) bool {
	if c.done == nil {
		c.done = map[*ssa.BasicBlock]bool{}
	}

	if c.done[b] {
		return true
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

func (c *calledFrom) predsAndEqualTo(b *ssa.BasicBlock, o types.Object) bool {
	if c.done == nil {
		c.done = map[*ssa.BasicBlock]bool{}
	}

	if c.done[b] {
		return true
	}
	c.done[b] = true

	if len(b.Preds) == 0 {
		return false
	}

	for _, p := range b.Preds {
		if !c.instrs(p.Instrs) {
			if !c.predsAndEqualTo(p, o) {
				return false
			}
			continue
		}

		// The function is definitely called in this pass.
		// Ideally we should check all the call of function,
		// but function is called once in almost all the cases.
		i := c.calledIndex(p.Instrs)
		ret, ok := p.Instrs[i].(ssa.Value)
		if !ok {
			return false
		}

		var compared bool
		for _, rr := range *ret.Referrers() {
			bo, ok := rr.(*ssa.BinOp)
			if !ok {
				continue
			}

			if bo.Op == token.EQL {
				ifi := getIfInstRefferringVal(bo)
				if ifi == nil {
					continue
				}
				eqPath := ifi.Block().Succs[0]
				if c.start != eqPath && !isASuccOf(c.start, eqPath) {
					continue
				}
				if bo.X == ret {
					if isSame(o, bo.Y) {
						compared = true
						break
					}
				}
				if bo.Y == ret {
					if isSame(o, bo.X) {
						compared = true
						break
					}
				}
			}

			if bo.Op == token.NEQ {
				ifi := getIfInstRefferringVal(bo)
				if ifi == nil {
					continue
				}
				eqPath := ifi.Block().Succs[1]
				if c.start != eqPath && !isASuccOf(c.start, eqPath) {
					continue
				}
				if bo.X == ret {
					if isSame(o, bo.Y) {
						compared = true
						break
					}
				}
				if bo.Y == ret {
					if isSame(o, bo.X) {
						compared = true
						break
					}
				}
			}
		}

		if !compared {
			return false
		}
	}

	return true
}

func isASuccOf(b *ssa.BasicBlock, p *ssa.BasicBlock, visited ...*ssa.BasicBlock) bool {
	for _, v := range visited {
		if v == p {
			return false
		}
	}
	for _, s := range p.Succs {
		if s == b {
			return true
		}
		v2 := append(visited, p)
		if isASuccOf(b, s, v2...) {
			return true
		}
	}
	return false
}

func getIfInstRefferringVal(v ssa.Value) *ssa.If {
	if v == nil {
		return nil
	}

	// Get the first IfInstruction referring v.
	// Ideally we should check all the refferres,
	// but the same value is used once in almost
	// all the cases.
	for _, br := range *v.Referrers() {
		if i, ok := br.(*ssa.If); ok {
			return i
		}
	}
	return nil
}

func binopReferredByIf(bo *ssa.BinOp, ifi *ssa.If) bool {
	if bo == nil || ifi == nil {
		return false
	}
	for _, br := range *bo.Referrers() {
		if br == ifi {
			return true
		}
	}
	return false
}

func isSame(o types.Object, oc ssa.Value) bool {
	// If pointer value is indirected, get the raw value.
	if ocu, ok := oc.(*ssa.UnOp); ok && ocu.Op == token.MUL {
		oc = ocu.X
	}

	switch oct := oc.(type) {
	case ssa.Member:
		if oct.Object().Id() == o.Id() {
			return true
		}
	case *ssa.Const:
		if c, ok := o.(*types.Const); ok {
			if c.Val() == oct.Value {
				return true
			}
		}
	}
	return false
}

func (c *CalledChecker) BeforeAndEqualTo(b *ssa.BasicBlock, receiver ssa.Value, method *types.Func, o types.Object) bool {
	// If pointer value is indirected, get the raw value.
	if ru, ok := receiver.(*ssa.UnOp); ok && ru.Op == token.MUL {
		receiver = ru.X
	}

	from := &calledFrom{recv: receiver, fs: []*types.Func{method}, ignore: c.Ignore, start: b}

	return from.predsAndEqualTo(b, o)
}

// CalledFromBefore checks whether receiver's method is called in an instruction
// which belongs to before i-th instructions, or in preds blocks of b.
// The first result is above value.
// The second result is whether type of i-th instruction does not much receiver
// or matches with ignore cases.
func CalledFromBefore(b *ssa.BasicBlock, i int, receiver ssa.Value, methods ...*types.Func) (called, ok bool) {
	return new(CalledChecker).FromBefore(b, i, receiver, methods...)
}

func CalledFromAfter(b *ssa.BasicBlock, i int, receiver ssa.Value, methods ...*types.Func) (called, ok bool) {
	return new(CalledChecker).FromAfter(b, i, receiver, methods...)
}

func CalledBeforeAndEqualTo(b *ssa.BasicBlock, receiver ssa.Value, method *types.Func, o types.Object) bool {
	return new(CalledChecker).BeforeAndEqualTo(b, receiver, method, o)
}
