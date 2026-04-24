// asmcheck

// Copyright 2023 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package codegen

func combine2string(p *[2]string, a, b string) {
	// amd64:`.*runtime[.]gcWriteBarrier4\(SB\)`
	// arm64:`.*runtime[.]gcWriteBarrier4\(SB\)`
	p[0] = a
	// amd64:-`.*runtime[.]gcWriteBarrier`
	// arm64:-`.*runtime[.]gcWriteBarrier`
	p[1] = b
}

func combine4string(p *[4]string, a, b, c, d string) {
	// amd64:`.*runtime[.]gcWriteBarrier8\(SB\)`
	// arm64:`.*runtime[.]gcWriteBarrier8\(SB\)`
	p[0] = a
	// amd64:-`.*runtime[.]gcWriteBarrier`
	// arm64:-`.*runtime[.]gcWriteBarrier`
	p[1] = b
	// amd64:-`.*runtime[.]gcWriteBarrier`
	// arm64:-`.*runtime[.]gcWriteBarrier`
	p[2] = c
	// amd64:-`.*runtime[.]gcWriteBarrier`
	// arm64:-`.*runtime[.]gcWriteBarrier`
	p[3] = d
}

func combine2slice(p *[2][]byte, a, b []byte) {
	// amd64:`.*runtime[.]gcWriteBarrier4\(SB\)`
	// arm64:`.*runtime[.]gcWriteBarrier4\(SB\)`
	p[0] = a
	// amd64:-`.*runtime[.]gcWriteBarrier`
	// arm64:-`.*runtime[.]gcWriteBarrier`
	p[1] = b
}

func combine4slice(p *[4][]byte, a, b, c, d []byte) {
	// amd64:`.*runtime[.]gcWriteBarrier8\(SB\)`
	// arm64:`.*runtime[.]gcWriteBarrier8\(SB\)`
	p[0] = a
	// amd64:-`.*runtime[.]gcWriteBarrier`
	// arm64:-`.*runtime[.]gcWriteBarrier`
	p[1] = b
	// amd64:-`.*runtime[.]gcWriteBarrier`
	// arm64:-`.*runtime[.]gcWriteBarrier`
	p[2] = c
	// amd64:-`.*runtime[.]gcWriteBarrier`
	// arm64:-`.*runtime[.]gcWriteBarrier`
	p[3] = d
}

func trickyWriteNil(p *int, q **int) {
	if p == nil {
		// We change "= p" to "= 0" in the prove pass, which
		// means we have one less pointer that needs to go
		// into the write barrier buffer.
		// amd64:`.*runtime[.]gcWriteBarrier1`
		*q = p
	}
}

type S struct {
	a, b string
	c    *int
}

var g1, g2 *int

func issue71228(dst *S, ptr *int) {
	// Make sure that the non-write-barrier write.
	// "sp.c = ptr" happens before the large write
	// barrier "*dst = *sp". We approximate testing
	// that by ensuring that two global variable write
	// barriers aren't combined.
	_ = *dst
	var s S
	sp := &s
	//amd64:`.*runtime[.]gcWriteBarrier1`
	g1 = nil
	sp.c = ptr // outside of any write barrier
	//amd64:`.*runtime[.]gcWriteBarrier1`
	g2 = nil
	//amd64:`.*runtime[.]wbMove`
	*dst = *sp
}

func writeDouble(p *[2]*int, x, y *int) {
	// arm64: `LDP ` `STP \(R[0-9]+, R[0-9]+\), \(`
	p[0] = x
	// arm64: `STP \(R[0-9]+, R[0-9]+\), 16\(`
	p[1] = y
}

func writeDoubleNil(p *[2]*int) {
	// arm64: `LDP ` `STP \(R[0-9]+, R[0-9]+\),` `STP \(ZR, ZR\),`
	p[0] = nil
	p[1] = nil
}

type wbNode struct {
	left  *wbNode
	right *wbNode
}

func wbNilCheckNewAlloc(val *wbNode) *wbNode {
	n := new(wbNode)
	// When storing a possibly-nil pointer to a newly allocated object,
	// the write barrier condition should include a nil check on val
	// (via ANDL combining the WB flag test with a pointer nil test),
	// so that gcWriteBarrier1 is skipped entirely when val is nil.
	// amd64:`TESTQ`
	// amd64:`ANDL`
	// amd64:`.*runtime[.]gcWriteBarrier1\(SB\)`
	n.left = val
	return n
}

func wbNoNilCheckExisting(n *wbNode, val *wbNode) {
	// When storing to an existing (non-newly-allocated) object, both
	// src and dst pointers need WB recording. The nil check optimization
	// should NOT apply here — no ANDL combining conditions.
	// amd64:-`ANDL`
	// amd64:`.*runtime[.]gcWriteBarrier2\(SB\)`
	n.left = val
}
