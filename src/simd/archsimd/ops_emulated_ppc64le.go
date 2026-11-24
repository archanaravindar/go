// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build goexperiment.simd && ppc64le && !ppc64le.power9

package archsimd

func (x Int32x4) Add(y Int32x4) (z Int32x4) {
	for i := range z.vals {
		z.vals[i] = x.vals[i] + y.vals[i]
	}
	return
}

func LoadInt32x4Array(p *[4]int32) (z Int32x4) { z.vals = *p; return }

func (x Int32x4) StoreArray(p *[4]int32) { *p = x.vals }

func (x Uint32x4) Add(y Uint32x4) (z Uint32x4) {
	for i := range z.vals {
		z.vals[i] = x.vals[i] + y.vals[i]
	}
	return
}

func LoadUint32x4Array(p *[4]uint32) (z Uint32x4) { z.vals = *p; return }

func (x Uint32x4) StoreArray(p *[4]uint32) { *p = x.vals }

func (x Int32x4) ToBits() (z Uint32x4) {
	for i := range z.vals {
		z.vals[i] = uint32(x.vals[i])
	}
	return
}
func (x Uint32x4) BitsToInt32() (z Int32x4) {
	for i := range z.vals {
		z.vals[i] = int32(x.vals[i])
	}
	return
}
func (x Int32x4) ConvertToUint32() (z Uint32x4) {
	for i := range z.vals {
		z.vals[i] = uint32(x.vals[i])
	}
	return
}
func (x Uint32x4) ConvertToInt32() (z Int32x4) {
	for i := range z.vals {
		z.vals[i] = int32(x.vals[i])
	}
	return
}

func (x Int64x2) Add(y Int64x2) (z Int64x2) {
	for i := range z.vals {
		z.vals[i] = x.vals[i] + y.vals[i]
	}
	return
}

func LoadInt64x2Array(p *[2]int64) (z Int64x2) { z.vals = *p; return }

func (x Int64x2) StoreArray(p *[2]int64) { *p = x.vals }

func (x Uint64x2) Add(y Uint64x2) (z Uint64x2) {
	for i := range z.vals {
		z.vals[i] = x.vals[i] + y.vals[i]
	}
	return
}

func LoadUint64x2Array(p *[2]uint64) (z Uint64x2) { z.vals = *p; return }

func (x Uint64x2) StoreArray(p *[2]uint64) { *p = x.vals }

func (x Int64x2) ToBits() (z Uint64x2) {
	for i := range z.vals {
		z.vals[i] = uint64(x.vals[i])
	}
	return
}
func (x Uint64x2) BitsToInt64() (z Int64x2) {
	for i := range z.vals {
		z.vals[i] = int64(x.vals[i])
	}
	return
}
func (x Int64x2) ConvertToUint64() (z Uint64x2) {
	for i := range z.vals {
		z.vals[i] = uint64(x.vals[i])
	}
	return
}
func (x Uint64x2) ConvertToInt64() (z Int64x2) {
	for i := range z.vals {
		z.vals[i] = int64(x.vals[i])
	}
	return
}

func (x Uint32x4) ReshapeToUint64s() (z Uint64x2) {
	z.vals[0] = uint64(x.vals[0]) | uint64(x.vals[1])<<32
	z.vals[1] = uint64(x.vals[2]) | uint64(x.vals[3])<<32
	return
}
func (x Uint64x2) ReshapeToUint32s() (z Uint32x4) {
	z.vals[0] = uint32(x.vals[0])
	z.vals[1] = uint32(x.vals[0] >> 32)
	z.vals[2] = uint32(x.vals[1])
	z.vals[3] = uint32(x.vals[1] >> 32)
	return
}
