// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build goexperiment.simd && ppc64le

package simd_test

import (
	"simd/archsimd"
	"testing"
)

// The ppc64le SIMD support currently implements Int32x4/Uint32x4 Add
// (VADDUWM), the array load/store (LXV/STXV on POWER9+, LXVD2X/STXVD2X on
// POWER8), and the no-op reinterpret/convert helpers. These tests exercise
// exactly that surface.

func TestInt32x4Add(t *testing.T) {
	a := [4]int32{1, 2, 3, 4}
	b := [4]int32{10, 20, 30, 40}
	want := []int32{11, 22, 33, 44}

	var got [4]int32
	archsimd.LoadInt32x4Array(&a).Add(archsimd.LoadInt32x4Array(&b)).StoreArray(&got)
	checkSlices(t, got[:], want)
}

func TestInt32x4AddNegative(t *testing.T) {
	a := [4]int32{-1, -2, 100, -2147483648}
	b := [4]int32{-1, 5, -100, -1}
	want := []int32{-2, 3, 0, 2147483647} // last lane wraps (two's complement)

	var got [4]int32
	archsimd.LoadInt32x4Array(&a).Add(archsimd.LoadInt32x4Array(&b)).StoreArray(&got)
	checkSlices(t, got[:], want)
}

func TestUint32x4Add(t *testing.T) {
	a := [4]uint32{1, 2, 0xFFFFFFFF, 0x80000000}
	b := [4]uint32{10, 20, 1, 0x80000000}
	want := []uint32{11, 22, 0, 0} // lanes 2,3 wrap modulo 2^32

	var got [4]uint32
	archsimd.LoadUint32x4Array(&a).Add(archsimd.LoadUint32x4Array(&b)).StoreArray(&got)
	checkSlices(t, got[:], want)
}

// TestInt32x4LoadStoreRoundTrip exercises the vector load+store path (LXV/STXV)
// on its own, with no arithmetic in between.
func TestInt32x4LoadStoreRoundTrip(t *testing.T) {
	a := [4]int32{-5, 0, 7, 123456}
	var got [4]int32
	archsimd.LoadInt32x4Array(&a).StoreArray(&got)
	checkSlices(t, got[:], a[:])
}

//go:noinline
func addInt32x4Noinline(a, b archsimd.Int32x4) archsimd.Int32x4 { return a.Add(b) }

// TestInt32x4AddAcrossCall passes vectors through a real (non-inlined) ABI
// boundary, exercising the register/stack passing that size.go's simdify decides
// for 128-bit SIMD types.
func TestInt32x4AddAcrossCall(t *testing.T) {
	a := [4]int32{1, 2, 3, 4}
	b := [4]int32{5, 6, 7, 8}
	want := []int32{6, 8, 10, 12}

	var got [4]int32
	addInt32x4Noinline(archsimd.LoadInt32x4Array(&a), archsimd.LoadInt32x4Array(&b)).StoreArray(&got)
	checkSlices(t, got[:], want)
}

//go:noinline
func blackhole() {}

// TestInt32x4AddSpill keeps a vector live across a call, forcing the register
// allocator to spill and reload it. On POWER9+ this is the D-form STXV/LXV to a
// 16-byte-aligned stack slot (the alignment guaranteed by simdify setting
// st.align = 16); on POWER8 it is the indexed STXVD2X/LXVD2X.
func TestInt32x4AddSpill(t *testing.T) {
	a := [4]int32{1, 2, 3, 4}
	b := [4]int32{100, 200, 300, 400}
	want := []int32{101, 202, 303, 404}

	sum := archsimd.LoadInt32x4Array(&a).Add(archsimd.LoadInt32x4Array(&b))
	blackhole() // clobbers caller-saved regs; sum must survive via a spill
	var got [4]int32
	sum.StoreArray(&got)
	checkSlices(t, got[:], want)
}

// TestReinterpret checks the no-op bit reinterpret/convert helpers round-trip to
// the identity (int32 and uint32 share a bit pattern, so these emit no code).
func TestReinterpret(t *testing.T) {
	a := [4]int32{-1, 0, 2147483647, -2147483648}
	var got [4]int32
	archsimd.LoadInt32x4Array(&a).ToBits().BitsToInt32().StoreArray(&got)
	checkSlices(t, got[:], a[:])

	var got2 [4]int32
	archsimd.LoadInt32x4Array(&a).ConvertToUint32().ConvertToInt32().StoreArray(&got2)
	checkSlices(t, got2[:], a[:])
}

func TestLen(t *testing.T) {
	var x archsimd.Int32x4
	var y archsimd.Uint32x4
	if x.Len() != 4 || y.Len() != 4 {
		t.Errorf("Len: got %d/%d, want 4/4", x.Len(), y.Len())
	}
}

var int32x4Sink archsimd.Int32x4

// TestArrayNilChecks confirms the load/store array intrinsics nil-check arg0.
func TestArrayNilChecks(t *testing.T) {
	t.Run("load", func(t *testing.T) {
		expectPanicPPC64(t, func() {
			int32x4Sink = archsimd.LoadInt32x4Array(nil)
		})
	})
	t.Run("store", func(t *testing.T) {
		expectPanicPPC64(t, func() {
			var a [4]int32
			archsimd.LoadInt32x4Array(&a).StoreArray(nil)
		})
	})
}

func expectPanicPPC64(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Error("did not panic")
		}
	}()
	f()
}
