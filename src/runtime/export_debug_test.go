// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build (amd64 || arm64 || ppc64le) && linux

package runtime

import (
	"internal/abi"
	"unsafe"
)

// InjectDebugCall injects a debugger call to fn into g. regArgs must
// contain any arguments to fn that are passed in registers, according
// to the internal Go ABI. It may be nil if no arguments are passed in
// registers to fn. args must be a pointer to a valid call frame (including
// arguments and return space) for fn, or nil. tkill must be a function that
// will send SIGTRAP to thread ID tid. gp must be locked to its OS thread and
// running.
//
// On success, InjectDebugCall returns the panic value of fn or nil.
// If fn did not panic, its results will be available in args.
func InjectDebugCall(gp *g, fn any, regArgs *abi.RegArgs, stackArgs any, tkill func(tid int) error, returnOnUnsafePoint bool) (any, error) {
	if gp.lockedm == 0 {
		return nil, plainError("goroutine not locked to thread")
	}

	tid := int(gp.lockedm.ptr().procid)
	if tid == 0 {
		return nil, plainError("missing tid")
	}

	f := efaceOf(&fn)
	if f._type == nil || f._type.Kind_&kindMask != kindFunc {
		return nil, plainError("fn must be a function")
	}
	fv := (*funcval)(f.data)

	a := efaceOf(&stackArgs)
	if a._type != nil && a._type.Kind_&kindMask != kindPtr {
		return nil, plainError("args must be a pointer or nil")
	}
	argp := a.data
	var argSize uintptr
	if argp != nil {
		argSize = (*ptrtype)(unsafe.Pointer(a._type)).Elem.Size_
	}

	h := new(debugCallHandler)
	h.gp = gp
	// gp may not be running right now, but we can still get the M
	// it will run on since it's locked.
	h.mp = gp.lockedm.ptr()
	h.fv, h.regArgs, h.argp, h.argSize = fv, regArgs, argp, argSize
	println("coming here arg size: ", argSize)
	//println("fv.fn" ,hex(fv.fn))
	////println("fv fv.fn  ", hex((uint64)fv), hex((uint64)fv.fn))
	////println("fv fv.fn regArgs argp ", hex((uint64)fv), hex((uint64)fv.fn),hex(uint64(regArgs)), hex(uint64(argp)))
	h.handleF = h.handle // Avoid allocating closure during signal

	defer func() { testSigtrap = nil }()
	for i := 0; ; i++ {
		testSigtrap = h.inject
		noteclear(&h.done)
		h.err = ""

		if err := tkill(tid); err != nil {
			return nil, err
		}
		// Wait for completion.
		notetsleepg(&h.done, -1)
		println("Output of inject call", h.err)
		if h.err != "" {
			switch h.err {
			case "call not at safe point":
				if returnOnUnsafePoint {
					// This is for TestDebugCallUnsafePoint.
					return nil, h.err
				}
				fallthrough
			case "retry _Grunnable", "executing on Go runtime stack", "call from within the Go runtime":
				// These are transient states. Try to get out of them.
				if i < 100 {
					usleep(100)
					Gosched()
					continue
				}
			}
			return nil, h.err
		}
		return h.panic, nil
	}
}

type debugCallHandler struct {
	gp      *g
	mp      *m
	fv      *funcval
	regArgs *abi.RegArgs
	argp    unsafe.Pointer
	argSize uintptr
	panic   any

	handleF func(info *siginfo, ctxt *sigctxt, gp2 *g) bool

	err     plainError
	done    note
	sigCtxt sigContext
}

func (h *debugCallHandler) inject(info *siginfo, ctxt *sigctxt, gp2 *g) bool {
	// TODO(49370): This code is riddled with write barriers, but called from
	// a signal handler. Add the go:nowritebarrierrec annotation and restructure
	// this to avoid write barriers.

	switch h.gp.atomicstatus.Load() {
	case _Grunning:
		if getg().m != h.mp {
			//println("trap on wrong M", getg().m, h.mp)
			return false
		}
		// Save the signal context
		h.saveSigContext(ctxt)
		// Set PC to debugCallV2.
		ctxt.setsigpc(uint64(abi.FuncPCABIInternal(debugCallV2)))
		//println("debugCallV2 pc ",hex(ctxt.pc())," switching to debug call protocol")
		// Call injected. Switch to the debugCall protocol.
		testSigtrap = h.handleF
	case _Grunnable:
		// Ask InjectDebugCall to pause for a bit and then try
		// again to interrupt this goroutine.
		//println("pause and retry")
		h.err = plainError("retry _Grunnable")
		notewakeup(&h.done)
	default:
		//println("unexpected state at call inject")
		h.err = plainError("goroutine in unexpected state at call inject")
		notewakeup(&h.done)
	}
	// Resume execution.
	return true
}

func (h *debugCallHandler) handle(info *siginfo, ctxt *sigctxt, gp2 *g) bool {
	// TODO(49370): This code is riddled with write barriers, but called from
	// a signal handler. Add the go:nowritebarrierrec annotation and restructure
	// this to avoid write barriers.

	// Double-check m.
	if getg().m != h.mp {
		//println("trap on wrong M", getg().m, h.mp)
		return false
	}
	f := findfunc(ctxt.sigpc())
	if !(hasPrefix(funcname(f), "runtime.debugCall") || hasPrefix(funcname(f), "debugCall")) {
		//println("trap in unknown function", funcname(f))
		return false
	}
	if !sigctxtAtTrapInstruction(ctxt) {
		//println("trap at non trap word PC =", hex(ctxt.sigpc()))
		return false
	}

	//println("Begin switch")
	switch status := sigctxtStatus(ctxt); status {
	case 0:
		// Frame is ready. Copy the arguments to the frame and to registers.
		// Call the debug function.
		//println("case 0 debugCallRun")
		h.debugCallRun(ctxt)
	case 1:
		// Function returned. Copy frame and result registers back out.
		//println("case 1 debugCallReturn")
		h.debugCallReturn(ctxt)
	case 2:
		// Function panicked. Copy panic out.
		//println("f now",funcname(f))
		//println("case 2 debugCallPanicked at PC",hex(ctxt.pc()))
		h.debugCallPanicOut(ctxt)
//	case 24:
//		//println("haha i m here")
	case 8:
		// Call isn't safe. Get the reason.
		h.debugCallUnsafe(ctxt)
		//println("case 8 debugCallUnsafe")
		//println("unsafe func ", funcname(f)," PC =", hex(ctxt.sigpc()))
		// Don't wake h.done. We need to transition to status 16 first.
	case 22:
		println("called debugcall* successfully")
	//	h.restoreSigContext(ctxt)
	//	notewakeup(&h.done)
	case 16:
		//println("case 16 restore sig context")
		h.restoreSigContext(ctxt)
		// Done
		notewakeup(&h.done)
	default:
		//println("unexpected debugCallV2 error")
		h.err = plainError("unexpected debugCallV2 status")
		notewakeup(&h.done)
	}
	
	//println("end switch")
	// Resume execution.
	return true
}
