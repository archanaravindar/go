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
	sp -= 2 * goarch.PtrSize
	ctxt.set_sp(sp)
	*(*uint64)(unsafe.Pointer(uintptr(sp))) = ctxt.link() // save the current lr
	ctxt.set_link(ctxt.pc())                              // set new lr to the current pc
	// Write the argument frame size.
	*(*uintptr)(unsafe.Pointer(uintptr(sp - 16))) = h.argSize
	// Save current registers.
	h.sigCtxt.savedRegs = *ctxt.cregs()
}

// case 0
func (h *debugCallHandler) debugCallRun(ctxt *sigctxt) {
	sp := ctxt.sp()
	memmove(unsafe.Pointer(uintptr(sp)+8), h.argp, h.argSize)
	if h.regArgs != nil {
		storeRegArgs(ctxt.cregs(), h.regArgs)
	}
	// Push return PC, which should be the signal PC+4, because
	// the signal PC is the PC of the trap instruction itself.
	ctxt.set_link(ctxt.pc() + 4)
	// Set PC to call and context register.
	ctxt.set_pc(uint64(h.fv.fn))
	sigctxtSetContextRegister(ctxt, uint64(uintptr(unsafe.Pointer(h.fv))))
}

// case 1
func (h *debugCallHandler) debugCallReturn(ctxt *sigctxt) {
	sp := ctxt.sp()
	memmove(h.argp, unsafe.Pointer(uintptr(sp)+8), h.argSize)
	if h.regArgs != nil {
		loadRegArgs(h.regArgs, ctxt.cregs())
	}
	// Restore the old lr from *sp
	olr := *(*uint64)(unsafe.Pointer(uintptr(sp)))
	ctxt.set_link(olr)
	pc := ctxt.pc()
	ctxt.set_pc(pc + 4) // step to next instruction
}

// case 2
func (h *debugCallHandler) debugCallPanicOut(ctxt *sigctxt) {
	sp := ctxt.sp()
	memmove(unsafe.Pointer(&h.panic), unsafe.Pointer(uintptr(sp)+8), 2*goarch.PtrSize)
	ctxt.set_pc(ctxt.pc() + 4)
}

// case 8
func (h *debugCallHandler) debugCallUnsafe(ctxt *sigctxt) {
	sp := ctxt.sp()
	reason := *(*string)(unsafe.Pointer(uintptr(sp) + 8))
	h.err = plainError(reason)
	ctxt.set_pc(ctxt.pc() + 4)
}

// case 16
func (h *debugCallHandler) restoreSigContext(ctxt *sigctxt) {
	// Restore all registers except for pc and sp
	pc, sp := ctxt.pc(), ctxt.sp()
	*ctxt.cregs() = h.sigCtxt.savedRegs
	ctxt.set_pc(pc + 4)
	ctxt.set_sp(sp)
}

// storeRegArgs sets up argument registers in the signal
// context state from an abi.RegArgs.
//
// Both src and dst must be non-nil.
func storeRegArgs(dst *sigcontext, src *abi.RegArgs) {
	for i, r := range src.Ints {
		dst.gp_regs[i] = uint64(r)
	}
	for i, r := range src.Floats {
		dst.fp_regs[i] = float64(r)
	}
}

func loadRegArgs(dst *abi.RegArgs, src *sigcontext) {
	for i := range dst.Ints {
		dst.Ints[i] = uintptr(src.gp_regs[i])
	}
	for i := range dst.Floats {
		dst.Floats[i] = uint64(src.fp_regs[i])
	}
}
