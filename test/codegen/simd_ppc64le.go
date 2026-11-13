// asmcheck

// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// These tests check PPC64LE SIMD code generation. SIMD on ppc64le requires
// POWER9 or later, so all checks use the power9 variant.

//go:build goexperiment.simd && ppc64le

package codegen

import (
	"simd/archsimd"
)

func addInt32x4(x, y archsimd.Int32x4) archsimd.Int32x4 {
	// ppc64le/power9:`VADDUWM`
	return x.Add(y)
}

func addUint32x4(x, y archsimd.Uint32x4) archsimd.Uint32x4 {
	// ppc64le/power9:`VADDUWM`
	return x.Add(y)
}

// On POWER9+ the 128-bit load/store use the single-instruction D-form
// LXV/STXV.
func loadStoreInt32x4(a, b *[4]int32) {
	// ppc64le/power9:`LXV`
	v := archsimd.LoadInt32x4Array(a)
	// ppc64le/power9:`STXV`
	v.StoreArray(b)
}

//go:noinline
func forceSpillPPC64() {}

// A vector kept live across a call is spilled then reloaded. On POWER9+ this is
// the D-form STXV/LXV to a 16-byte-aligned stack slot.
func spillAroundCallInt32x4(x archsimd.Int32x4) archsimd.Int32x4 {
	forceSpillPPC64()
	// ppc64le/power9:`STXV` `LXV`
	return x
}
