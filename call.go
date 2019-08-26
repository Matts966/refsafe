package refsafe

import (
	"go/token"
	"go/types"
	"log"

	"golang.org/x/tools/go/ssa"
)

// CalledChecker checks a function is called.
// See From and Func.
type CalledChecker struct {
	Ignore func(instr ssa.Instruction) bool
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

	if f.Name() == "open" && fn.Name() == "open" {
		// log.Printf("f: %#v\n", f)
		// log.Printf("fn: %#v\n", fn)
		// log.Printf("f == fn: %v\n", f == fn)
		// log.Printf("Args: %#v\n", common.Args[0])
		// log.Printf("recv: %#v\n", recv)
		// log.Printf("Args[0].ref%#v\n", *(common.Args[0].Referrers()))
		// log.Printf("recv.ref: %#v", *(recv.Referrers()))
		// log.Printf("args.ref: %#v", (*(common.Args[0].Referrers()))[0])
		// log.Printf("recv: %#v", recv)
	}

	// log.Println(common.Args[0])

	if recv != nil &&
		common.Signature().Recv() != nil &&
		(len(common.Args) == 0 && recv != nil || common.Args[0] != recv &&
			!isReferrer(recv, common.Args[0])) {
		// recv not in common.Args[0].Referrers() && common.Args[0] not in common.Referrers()
		// log.Println(isReferrer(recv, common.Args[0]))
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

	// log.Println(v)
	// log.Println(receiver)

	// ここもいらない説。。。わからん。
	// if !identical(v.Type(), receiver) {
	// 	return false, false
	// }

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
			// log.Println(Called(instr, c.recv, f))
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

// CalledFromAfter is an alias to CalledFrom to distinguish CalledFromBefore from
// CalledFrom.
var CalledFromAfter = CalledFrom

// Called returns true when f is called in the instr.
// If recv is not nil, Called also checks the receiver.
func Called(instr ssa.Instruction, recv ssa.Value, f *types.Func) bool {
	return new(CalledChecker).Func(instr, recv, f)
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

	log.Println(v.Type())

	// log.Printf("%#v\n", b.Instrs[i-1])
	// log.Printf("%#v\n", b.Instrs[i])
	// log.Printf("%#v\n", b.Instrs[i+1])

	// Call の typeが返り値の場合かなえらずfalse?
	// if !identical(v.Type(), receiver) {
	// 	return false, false
	// }

	from := &calledFrom{recv: v, fs: methods, ignore: c.Ignore}

	// log.Println(from.instrs(b.Instrs[:i]))
	// log.Println(from.preds(b))

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
// which belongs to before i-th instructions, or in preds blocks of b.
// The first result is above value.
// The second result is whether type of i-th instruction does not much receiver
// or matches with ignore cases.
func CalledFromBefore(b *ssa.BasicBlock, i int, receiver ssa.Value, methods ...*types.Func) (called, ok bool) {
	return new(CalledChecker).FromBefore(b, i, receiver, methods...)
}
