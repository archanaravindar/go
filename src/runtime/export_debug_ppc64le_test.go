// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ppc64le && linux

package runtime

import (
	"internal/abi"
	"internal/goarch"
	"unsafe"
)

type sigContext struct {
	savedRegs sigcontext
}

func sigctxtSetContextRegister(ctxt *sigctxt, x uint64) {
	ctxt.regs().gpr[11] = x
}

func sigctxtAtTrapInstruction(ctxt *sigctxt) bool {
	return *(*uint32)(unsafe.Pointer(ctxt.sigpc())) == 0x7fe00008 // Trap
}

func sigctxtStatus(ctxt *sigctxt) uint64 {
	return ctxt.r20()
}

func (h *debugCallHandler) saveSigContext(ctxt *sigctxt) {
	sp := ctxt.sp()
	sp -= 3 * goarch.PtrSize
	ctxt.set_sp(sp)
	*(*uint64)(unsafe.Pointer(uintptr(sp))) = ctxt.link() // save the current lr
	ctxt.set_link(ctxt.pc())                              // set new lr to the current pc
	// Write the argument frame size.
	println("sp addr at save sig context: ", hex((uintptr)(sp)), "pc addr at save sign context ",hex((uintptr)(ctxt.pc())))
dumpregs(ctxt)
	*(*uintptr)(unsafe.Pointer(uintptr(sp - 24))) = h.argSize
	println("arg size inside savecontext",h.argSize)
	println("arg size stored at  : ", hex((uintptr)(sp-24)))
	// Save current registers.
	h.sigCtxt.savedRegs = *ctxt.cregs()
}

// case 0
func (h *debugCallHandler) debugCallRun(ctxt *sigctxt) {
	sp := ctxt.sp()
	println("sp addr at save sig context: ", hex((uintptr)(sp)), "pc addr at save sign context ",hex((uintptr)(ctxt.pc())))
dumpregs(ctxt)
	memmove(unsafe.Pointer(uintptr(sp)+8), h.argp, h.argSize)
	if h.regArgs != nil {
println("storing reg args at ",hex(uintptr(sp)+8),", memmove size ", h.argSize)
		storeRegArgs(ctxt.cregs(), h.regArgs)
	}
	// Push return PC, which should be the signal PC+4, because
	// the signal PC is the PC of the trap instruction itself.
	ctxt.set_link(ctxt.pc() + 4)
	// Set PC to call and context register.
println("setting pc of func call ", hex(uint64(h.fv.fn)))
	ctxt.set_pc(uint64(h.fv.fn))
	sigctxtSetContextRegister(ctxt, uint64(uintptr(unsafe.Pointer(h.fv))))
}

// case 1
func (h *debugCallHandler) debugCallReturn(ctxt *sigctxt) {
	println("inside debugCallReturn")
	sp := ctxt.sp()
	memmove(h.argp, unsafe.Pointer(uintptr(sp)+8), h.argSize)
	if h.regArgs != nil {
		loadRegArgs(h.regArgs, ctxt.cregs())
	}
dumpregs(ctxt)
	// Restore the old lr from *sp
	olr := *(*uint64)(unsafe.Pointer(uintptr(sp)))
	ctxt.set_link(olr)
	pc := ctxt.pc()
	ctxt.set_pc(pc + 4) // step to next instruction
}

// case 2
func (h *debugCallHandler) debugCallPanicOut(ctxt *sigctxt) {
	sp := ctxt.sp()
	println("debug call panic", hex(ctxt.pc()), hex(sp))
	memmove(unsafe.Pointer(&h.panic), unsafe.Pointer(uintptr(sp)+8), 2*goarch.PtrSize)
	ctxt.set_pc(ctxt.pc() + 4)
}

// case 8
func (h *debugCallHandler) debugCallUnsafe(ctxt *sigctxt) {
	sp := ctxt.sp()
	println("debug call unsafe", hex(ctxt.pc()), hex(sp))
	reason := *(*string)(unsafe.Pointer(uintptr(sp) + 8))
	h.err = plainError(reason)
	ctxt.set_pc(ctxt.pc() + 4)
}

// case 16
func (h *debugCallHandler) restoreSigContext(ctxt *sigctxt) {
	// Restore all registers except for pc and sp
	pc, sp := ctxt.pc(), ctxt.sp()
	println("inside restore sig context")
	dumpregs(ctxt)
	*ctxt.cregs() = h.sigCtxt.savedRegs
	println("after copy of saved regs")
	dumpregs(ctxt)
	println("before set PC and sp values", hex(ctxt.pc()), hex(ctxt.sp()))
	ctxt.set_pc(pc+4)
	ctxt.set_sp(sp)
	println("PC and sp values", hex(ctxt.pc()), hex(ctxt.sp()))
}

// storeRegArgs sets up argument registers in the signal
// context state from an abi.RegArgs.
//
// Both src and dst must be non-nil.
func storeRegArgs(dst *sigcontext, src *abi.RegArgs) {
	println("inside storeRegArgs")
	// Gprs R3..R10 are used to pass int arguments in registers on PPC64
	for i, r := range [8]int{} {
		println("saving gpr reg ",i+3)
		dst.gp_regs[i+3] = uint64(src.Ints[r])
	}
	// Fprs F1..F13 are used to pass float arguments in registers on PPC64
	//for i, r := range[13]int{} {
	//	println("saving fpr reg ",i+1)
//		dst.fp_regs[i+1] = src.Floats[r]
//	}
}

func loadRegArgs(dst *abi.RegArgs, src *sigcontext) {
	// Gprs R3..R10 are used to pass int arguments in registers on PPC64
	for i, r := range [8]int{} {
		dst.Ints[r] = uintptr(src.gp_regs[i+3])
	}
	// Fprs F1..F13 are used to pass float arguments in registers on PPC64
//	for i,r := range [13]int{} {
//		dst.Floats[r] = uint64(src.fp_regs[i+1])
//	}
}
